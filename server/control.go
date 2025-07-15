package server

import (
	"fmt"
	"io"
	"slices"

	"neofrp/common/config"
	C "neofrp/common/constant"
	"neofrp/common/multidialer"
	P "neofrp/common/protocol"
)

type ControlHandler struct {
	config *config.ServerConfig
	conn   multidialer.Stream
}

func NewControlHandler(config *config.ServerConfig, conn multidialer.Stream) *ControlHandler {
	return &ControlHandler{
		config: config,
		conn:   conn,
	}
}

func (h *ControlHandler) Handshake() error {
	buff := make([]byte, 1)
	io.ReadFull(h.conn, buff)
	// Verify the version number
	if buff[0] != C.Version {
		h.conn.Write([]byte{P.ReturnCodeOtherError})
		return fmt.Errorf("unsupported version: %d", buff[0])
	}
	// Validate the token
	io.ReadFull(h.conn, buff)
	n := uint8(buff[0])
	buff = make([]byte, n)
	io.ReadFull(h.conn, buff)
	if !slices.Contains(h.config.RecognizedTokens, string(buff)) {
		h.conn.Write([]byte{P.ReturnCodeUnrecognizedToken})
		return fmt.Errorf("unrecognized token: %s", string(buff))
	}
	return nil
}
