package http

import (
	"io"
	"net"
	"time"
)

type Connection struct {
	r *io.PipeReader
	w *io.PipeWriter
}

func newConn() *Connection {
	r, w := io.Pipe()
	return &Connection{r: r, w: w}
}

func (c *Connection) Read(b []byte) (n int, err error) {
	return c.r.Read(b)
}

func (c *Connection) Write(b []byte) (n int, err error) {
	return c.w.Write(b)
}

func (c *Connection) Close() error {
	return c.w.Close()
}

func (c *Connection) LocalAddr() net.Addr {
	return nil
}

func (c *Connection) RemoteAddr() net.Addr {
	return nil
}

func (c *Connection) SetDeadline(t time.Time) error {
	return nil
}

func (c *Connection) SetReadDeadline(t time.Time) error {
	return nil
}

func (c *Connection) SetWriteDeadline(t time.Time) error {
	return nil
}

var _ net.Conn = &Connection{}
