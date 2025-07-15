package constant

import (
	"bufio"
	"bytes"
	"io"
	"net"
)

// BufferReadWriteCloser is a struct that implements the io.ReadWriteCloser interface
type BufferReadWriteCloser struct {
	buf *bytes.Buffer
}

type ChannelIdentifier uint16
type ClientAuthTokenType = string
type TransportType byte
type PortType uint16

// BufferedConn wraps a net.Conn with a bufio.Reader to handle cases where
// data has been read into the buffer during connection type detection
type BufferedConn struct {
	Reader *bufio.Reader
	Conn   net.Conn
}

// BidirectionalPipe wraps io.Pipe to implement the ReadWriteCloser interface
type BidirectionalPipe struct {
	Reader *io.PipeReader
	Writer *io.PipeWriter
}
