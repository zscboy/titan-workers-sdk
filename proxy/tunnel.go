package proxy

import (
	"encoding/binary"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type CMD int

const (
	cMDNone              = 0
	cMDReqData           = 1
	cMDReqCreated        = 2
	cMDReqClientClosed   = 3
	cMDReqClientFinished = 4
	cMDReqServerFinished = 5
	cMDReqServerClosed   = 6
)

const maxCap = 100

type Tunnel struct {
	uuid   string
	idx    int
	tunmgr *TunMgr
	url    string
	cap    int
	conn   *websocket.Conn
	// lastActivitTime time.Time
	reqq      *Reqq
	writeLock sync.Mutex
	busy      int
}

func newTunnel(uuid string, idx int, tunmgr *TunMgr, url string, cap int) *Tunnel {
	return &Tunnel{
		uuid:      uuid,
		idx:       idx,
		tunmgr:    tunmgr,
		url:       url,
		cap:       cap,
		writeLock: sync.Mutex{},
		reqq:      newReqq(cap),
	}
}

func (t *Tunnel) connect() error {
	url := fmt.Sprintf("%s?cap=%d&uuid=%s", t.url, t.cap, t.uuid)
	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		return fmt.Errorf("dial %s failed %s", url, err.Error())
	}
	defer conn.Close()

	t.conn = conn

	log.Infof("new tun %s", url)

	defer t.onWebsocketClose()

	// conn.SetPongHandler(tc.onPone)
	// go tc.keepalive()

	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Info("Error reading message:", err)
			return err
		}

		if messageType != websocket.BinaryMessage {
			log.Errorf("unsupport message type %d", messageType)
			continue
		}

		if err = t.onTunnelMsg(p); err != nil {
			log.Errorf("onTunnelMsg: %s", err.Error())
		}
	}
}

func (t *Tunnel) sendPing() error {
	now := time.Now()
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf[0:], uint64(now.Unix()))
	return t.conn.WriteMessage(websocket.PingMessage, buf)
}

func (t *Tunnel) onWebsocketClose() {
	if t.conn == nil {
		return
	}
	t.tunmgr.onTunnelBroken(t)
}

func (t *Tunnel) resetBusy() {
	t.busy = 0
}

func (t *Tunnel) onTunnelMsg(message []byte) error {
	cmd := message[0]
	idx := binary.LittleEndian.Uint16(message[1:])
	tag := binary.LittleEndian.Uint16(message[3:])

	switch cmd {
	case uint8(cMDReqData):
		data := message[5:]
		return t.onServerRequestData(idx, tag, data)
	case uint8(cMDReqServerFinished):
		return t.onServerRecvFinish(idx, tag)
	case uint8(cMDReqServerClosed):
		t.onServerRecvClose(idx, tag)
	default:
		log.Errorf("[Tunnel]unknown cmd:", cmd)
	}

	return nil
}

func (t *Tunnel) onServerRequestData(idx, tag uint16, data []byte) error {
	log.Infof("onServerRequestData, idx:%d tag:%d, data len:%d", idx, tag, len(data))

	req := t.reqq.getReq(idx, tag)
	if req == nil {
		return fmt.Errorf("can not find request, idx %d, tag %d", idx, tag)
	}
	return req.write(data)
}

func (t *Tunnel) onServerRecvFinish(idx, tag uint16) error {
	log.Infof("onServerRequestFinish, idx:%d tag:%d", idx, tag)

	req := t.reqq.getReq(idx, tag)
	if req == nil {
		return fmt.Errorf("can not find request, idx %d, tag %d", idx, tag)
	}
	return req.onServerFinished()
}

func (t *Tunnel) onServerRecvClose(idx, tag uint16) {
	log.Infof("onServerRequestClose, idx:%d tag:%d", idx, tag)
	t.reqq.free(idx, tag)
}

func (t *Tunnel) onAcceptRequest(conn net.Conn, dest *DestAddr) error {
	log.Infof("onAcceptRequest, dest addr %s port %d", dest.Addr, dest.Port)
	req, err := t.acceptRequestInternal(conn, dest)
	if err != nil {
		return err
	}
	return t.serveConn(conn, req.idx, req.tag)
}

func (tm *TunMgr) OnAcceptHTTPsRequest(conn net.Conn, dest *DestAddr, header []byte) {
	// allocate tunnel for sock
	tun := tm.allocTunnelForRequest()
	if tun == nil {
		log.Errorf("[TunMgr] failed to alloc tunnel for sock, discard it")
		return
	}

	if err := tun.onAcceptHTTPsRequest(conn, dest, header); err != nil {
		log.Errorf("onAcceptHTTPRequest %s", err.Error())
	}
}

func (tm *TunMgr) OnAcceptHTTPRequest(conn net.Conn, dest *DestAddr, header []byte) {
	// allocate tunnel for sock
	tun := tm.allocTunnelForRequest()
	if tun == nil {
		log.Errorf("[TunMgr] failed to alloc tunnel for sock, discard it")
		return
	}

	if err := tun.onAcceptHTTPRequest(conn, dest, header); err != nil {
		log.Errorf("onAcceptHTTPRequest %s", err.Error())
	}
}

func (t *Tunnel) acceptRequestInternal(conn net.Conn, destAddr *DestAddr) (*Request, error) {
	if !t.isConnected() {
		return nil, fmt.Errorf("[Tunnel] accept sock failed, tunnel is disconnected")
	}

	req := t.reqq.allocReq(conn)
	if req == nil {
		return nil, fmt.Errorf("[Tunnel] allocReq failed, discard sock")
	}

	// send create message to server
	err := t.sendCreate2Server(req, destAddr)

	return req, err
}

func (t *Tunnel) serveConn(conn net.Conn, idx uint16, tag uint16) error {
	defer conn.Close()

	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			// log.Println("proxy read failed:", err)
			return t.onClientRecvClose(idx, tag)
		}

		if n == 0 {
			// log.Println("proxy read, server half close")
			return t.onClientRecvFinished(idx, tag)
		}

		t.onClientRecvData(idx, tag, buf[:n])
	}
}

func (t *Tunnel) serveHTTPRequest(conn net.Conn, idx uint16, tag uint16) error {
	buf := make([]byte, 4096)
	for {
		n, err := conn.Read(buf)
		if err != nil {
			// log.Println("proxy read failed:", err)
			return t.onClientRecvClose(idx, tag)
		}

		if n == 0 {
			// log.Println("proxy read, server half close")
			return t.onClientRecvFinished(idx, tag)
		}

		t.onClientRecvData(idx, tag, buf[:n])
	}
}

func (t *Tunnel) onClientRecvClose(idx, tag uint16) error {
	log.Infof("onClientClose idx:%d tag:%d", idx, tag)
	t.reqq.free(idx, tag)
	return t.sendCtl2Server(uint8(cMDReqClientClosed), idx, tag)
}

func (t *Tunnel) onClientRecvFinished(idx, tag uint16) error {
	log.Infof("onClientRecvFinished idx:%d tag:%d", idx, tag)
	if !t.reqq.reqValid(idx, tag) {
		return fmt.Errorf("[tunnel] connect is change")
	}

	// send to server
	return t.sendCtl2Server(uint8(cMDReqClientFinished), idx, tag)
}

func (t *Tunnel) onClientRecvData(idx, tag uint16, data []byte) error {
	log.Infof("onClientRecvData, idx %d tag %d", idx, tag)
	if !t.reqq.reqValid(idx, tag) {
		return fmt.Errorf("onClientRecvData, invalid idx %d tag %d", idx, tag)
	}

	buf := make([]byte, 5+len(data))
	buf[0] = byte(cMDReqData)
	binary.LittleEndian.PutUint16(buf[1:], idx)
	binary.LittleEndian.PutUint16(buf[3:], tag)
	copy(buf[5:], data)

	return t.write(buf)
}

func (t *Tunnel) sendCreate2Server(req *Request, destAddr *DestAddr) error {
	addrLength := len(destAddr.Addr)
	buf := make([]byte, 9+addrLength)

	// 1 byte cmd
	buf[0] = uint8(cMDReqCreated)

	// 2 bytes req_idx
	binary.LittleEndian.PutUint16(buf[1:], uint16(req.idx))

	// 2 bytes req_tag
	binary.LittleEndian.PutUint16(buf[3:], uint16(req.tag))

	// 1 byte address_type, always be domain type
	buf[5] = 1

	// 1 byte domain length
	buf[6] = byte(addrLength)

	// domain
	copy(buf[7:], []byte(destAddr.Addr))

	// 2 bytes req_tag
	offset := 7 + addrLength
	binary.LittleEndian.PutUint16(buf[offset:], uint16(destAddr.Port))

	if t.isConnected() {
		return t.write(buf)
	}

	return fmt.Errorf("[Tunnel] accept sock failed, tunnel is disconnected")
}

func (t *Tunnel) sendCtl2Server(cmd uint8, idx, tag uint16) error {
	buf := make([]byte, 5)
	buf[0] = cmd
	binary.LittleEndian.PutUint16(buf[1:], idx)
	binary.LittleEndian.PutUint16(buf[3:], tag)
	return t.write(buf)
}

func (t *Tunnel) write(data []byte) error {
	t.writeLock.Lock()
	defer t.writeLock.Unlock()
	return t.conn.WriteMessage(websocket.BinaryMessage, data)
}

func (t *Tunnel) getServiceID(r *http.Request) string {
	// path = /project/nodeID/project/{custom}
	reqPath := r.URL.Path
	parts := strings.Split(reqPath, "/")
	if len(parts) >= 4 {
		return parts[3]
	}
	return ""
}

func (t *Tunnel) getHeaderString(r *http.Request, serviceID string) string {
	headerString := fmt.Sprintf("%s %s %s\r\n", r.Method, r.URL.String(), r.Proto)
	headerString += fmt.Sprintf("Host: %s\r\n", r.RemoteAddr)
	for name, values := range r.Header {
		for _, value := range values {
			headerString += fmt.Sprintf("%s: %s\r\n", name, value)
		}
	}
	headerString += "\r\n"
	return headerString
}

func (t *Tunnel) isFulled() bool {
	return t.reqq.isFulled()
}

func (t *Tunnel) isConnected() bool {
	return t.conn != nil
}
