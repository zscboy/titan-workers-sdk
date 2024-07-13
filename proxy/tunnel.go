package proxy

import (
	"context"
	"encoding/binary"
	"fmt"
	"io"
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
	cMDPing              = 1
	cMDPong              = 2
	cMReqBegin           = 3
	cMDReqData           = 3
	cMDReqCreated        = 4
	cMDReqClientClosed   = 5
	cMDReqClientFinished = 6
	cMDReqServerFinished = 7
	cMDReqServerClosed   = 8
	cMDReqRefreshQuota   = 9
	cMDReqEnd            = 10
)

// const CMD_None = 0;
// const CMD_Ping = 1;
// const CMD_Pong = 2;
// const CMD_ReqBEGIN = 3;
// // client and server use this cmd to send request's data
// const CMD_ReqData = 3;
// // client notify server that a new request has created
// const CMD_ReqCreated = 4;
// // client notify server that a request has closed
// const CMD_ReqClientClosed = 5;
// // client notify server that a request has finished, but not closed
// const CMD_ReqClientFinished = 6;
// // server notify client that a request has finished, but not closed
// const CMD_ReqServerFinished = 7;
// // server notify client that a request has closed
// const CMD_ReqServerClosed = 8;
// // server notify client that a request quota has been refresh,
// // means that client can send more data of this request
// const CMD_ReqRefreshQuota = 9;
// const CMD_ReqEND = 10;

const maxCap = 100

type Tunnel struct {
	uuid   string
	idx    int
	tunmgr *TunMgr
	cap    int
	conn   *websocket.Conn
	// lastActivitTime time.Time
	reqq      *Reqq
	writeLock sync.Mutex
	busy      int
	url       string
	isDestroy bool
}

func newTunnel(uuid string, idx int, tunmgr *TunMgr, cap int, url string) *Tunnel {
	tun := &Tunnel{
		uuid:      uuid,
		idx:       idx,
		tunmgr:    tunmgr,
		cap:       cap,
		writeLock: sync.Mutex{},
		reqq:      newReqq(cap),
		url:       url,
	}

	if err := tun.connect(); err != nil {
		log.Warnf(" new turnnel faile %s", err.Error())
		tun.tunmgr.onTunnelBroken(tun)
	} else {
		go tun.serveWebsocket()
	}

	return tun
}

func (t *Tunnel) connect() error {
	url := fmt.Sprintf("%s?cap=%d&uuid=%s", t.url, t.cap, t.uuid)

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	conn, resp, err := websocket.DefaultDialer.DialContext(ctx, url, nil)
	if err != nil {
		if resp != nil {
			body, _ := io.ReadAll(resp.Body)
			return fmt.Errorf("dial %s failed %s, rsp: %s", url, err.Error(), string(body))
		}
		return fmt.Errorf("dial %s failed %s", url, err.Error())
	}
	t.conn = conn

	log.Infof("new tun %s", url)
	return nil
}

func (t *Tunnel) destroy() error {
	if t.conn != nil {
		return t.conn.Close()
	}
	t.isDestroy = true

	return nil
}

func (t *Tunnel) serveWebsocket() error {
	defer t.onWebsocketClose()
	conn := t.conn
	for {
		messageType, p, err := conn.ReadMessage()
		if err != nil {
			log.Errorf("Error reading message:", err)
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

func (t *Tunnel) onWebsocketClose() {
	if t.conn != nil {
		t.conn.Close()
		t.conn = nil
	}

	t.reqq.cleanup()

	t.tunmgr.onTunnelBroken(t)
}

func (t *Tunnel) reconnect() error {
	if err := t.connect(); err != nil {
		t.tunmgr.onTunnelBroken(t)
		return err
	}

	go t.serveWebsocket()
	return nil
}

func (t *Tunnel) sendPing() error {
	now := time.Now()
	buf := make([]byte, 9)
	buf[0] = byte(cMDPing)
	binary.LittleEndian.PutUint64(buf[1:], uint64(now.Unix()))
	return t.write(buf)
}

func (t *Tunnel) sendPong(data []byte) error {
	buf := make([]byte, len(data))
	buf[0] = byte(cMDPong)
	copy(buf[1:], data[1:])
	return t.write(buf)
}

func (t *Tunnel) resetBusy() {
	t.busy = 0
}

func (t *Tunnel) onTunnelMsg(message []byte) error {
	cmd := uint8(message[0])
	// log.Debugf("onTunnelMsg messag len %d, cmd %d", len(message), cmd)
	if t.isRequestCmd(cmd) {
		return t.onTunnelRequestMessage(cmd, message)
	}

	switch cmd {
	case cMDPing:
		// log.Debugf("onPing")
		return t.sendPong(message)
	case cMDPong:
		// log.Debugf("onPong")
		return t.onPong(message)

	default:
		log.Errorf("[Tunnel]unknown cmd:", cmd)
	}
	return nil
}

func (t *Tunnel) isRequestCmd(cmd uint8) bool {
	if cmd >= cMReqBegin && cmd < cMDReqEnd {
		return true
	}

	return false
}

func (t *Tunnel) onPong(message []byte) error {
	if len(message) != 9 {
		return fmt.Errorf("message len != 9")
	}
	return nil
}

func (t *Tunnel) onTunnelRequestMessage(cmd uint8, message []byte) error {

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
		// return fmt.Errorf("onServerRequestData can not find request, idx %d, tag %d", idx, tag)
		return nil
	}
	return req.write(data)
}

func (t *Tunnel) onServerRecvFinish(idx, tag uint16) error {
	log.Debugf("onServerRecvFinish, idx:%d tag:%d", idx, tag)

	req := t.reqq.getReq(idx, tag)
	if req == nil {
		return fmt.Errorf("onServerRecvFinish can not find request, idx %d, tag %d", idx, tag)
	}
	return req.onServerFinished()
}

func (t *Tunnel) onServerRecvClose(idx, tag uint16) {
	log.Debugf("onServerRecvClose, idx:%d tag:%d", idx, tag)
	t.reqq.free(idx, tag)
}

func (t *Tunnel) onAcceptRequest(conn net.Conn, dest *DestAddr) error {
	log.Debugf("onAcceptRequest, dest %s:%d", dest.Addr, dest.Port)

	req, err := t.acceptRequestInternal(conn, dest)
	if err != nil {
		return err
	}

	log.Debugf("onAcceptRequest, alloc idx %d tag %d", req.idx, req.tag)
	return t.serveConn(conn, req.idx, req.tag)
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
			// log.Debugf("serveConn: %s", err.Error())
			// if err == io.EOF {
			// 	return t.onClientRecvFinished(idx, tag)
			// }

			if !isNetErrUseOfCloseNetworkConnection(err) {
				return t.onClientRecvClose(idx, tag)
			}
			return nil
		}

		if n == 0 {
			// log.Println("proxy read, server half close")
			t.onClientRecvFinished(idx, tag)
			continue
		}

		t.onClientRecvData(idx, tag, buf[:n])
	}
}

func (t *Tunnel) serveHTTPRequest(conn net.Conn, idx uint16, tag uint16) error {
	return t.onClientRecvFinished(idx, tag)
}

func (t *Tunnel) onClientRecvClose(idx, tag uint16) error {
	log.Debugf("onClientClose idx:%d tag:%d", idx, tag)
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
	// log.Debugf("onClientRecvData, idx %d tag %d", idx, tag)
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

	return fmt.Errorf("[Tunnel] sendCreate2Server failed, tunnel is disconnected")
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

	if t.conn == nil {
		return fmt.Errorf("t.conn == nil ")
	}
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
