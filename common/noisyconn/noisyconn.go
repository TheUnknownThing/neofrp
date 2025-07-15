package noisyconn

import (
	"fmt"
	"io"
	"net"

	"neofrp/common/hexdump"

	"github.com/charmbracelet/log"
)

// NoisyConn is a wrapper around net.Conn that adds logging for all read and write operations.
type NoisyConn struct {
	net.Conn
}

func NewNoisyConn(conn net.Conn) *NoisyConn {
	return &NoisyConn{Conn: conn}
}

func (nc *NoisyConn) Read(b []byte) (n int, err error) {
	n, err = nc.Conn.Read(b)
	if n > 0 {
		log.Debug("NoisyConn: Read data from connection", "bytes", n)
		fmt.Print(hexdump.Dump(b[:n]))
	}
	if err != nil && err != io.EOF {
		log.Error("Error reading from connection", "error", err)
	}
	return n, err
}

func (nc *NoisyConn) Write(b []byte) (n int, err error) {
	n, err = nc.Conn.Write(b)
	if n > 0 {
		log.Debug("NoisyConn: Wrote data to connection", "bytes", n)
		fmt.Print(hexdump.Dump(b[:n]))
	}
	if err != nil {
		log.Error("NoisyConn: Error writing to connection", "error", err)
	}
	return n, err
}
