package proxy

import (
	"fmt"
	"net"
)

type Request struct {
	idx    uint16
	tag    uint16
	inused bool
	conn   net.Conn
}

func newRequest(idx uint16) *Request {
	return &Request{idx: idx}
}

func (r *Request) write(data []byte) error {
	if r.conn == nil {
		return fmt.Errorf("request idx %d, writer is nil", r.idx)
	}
	return r.writeAll(data)
}

func (r *Request) writeAll(buf []byte) error {
	wrote := 0
	l := len(buf)
	for {
		n, err := r.conn.Write(buf[wrote:])
		if err != nil {
			return err
		}

		wrote = wrote + n
		if wrote == l {
			break
		}
	}
	return nil
}

func (r *Request) onServerFinished() error {
	if r.conn != nil {
		tcpConn, ok := r.conn.(*net.TCPConn)
		if ok {
			return tcpConn.CloseWrite()
		}
	}
	return nil
}

func (r *Request) dofree() {
	if r.conn != nil {
		r.conn.Close()
		r.conn = nil
	}
}
