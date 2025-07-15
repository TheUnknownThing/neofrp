package client

import (
	E "errors"
	"io"
	"neofrp/common/multidialer"

	"neofrp/common/config"
	C "neofrp/common/constant"
	P "neofrp/common/protocol"
)

type ControlHandler struct {
	Config        *config.ClientConfig
	ControlStream multidialer.Stream
}

func NewControlHandler(config *config.ClientConfig, controlStream multidialer.Stream) *ControlHandler {
	return &ControlHandler{
		Config:        config,
		ControlStream: controlStream,
	}
}

func (h *ControlHandler) Handshake() error {
	msg := []byte{C.Version, byte(len(h.Config.Token))}
	msg = append(msg, []byte(h.Config.Token)...)
	_, err := h.ControlStream.Write(msg)
	if err != nil {
		return err
	}
	// Read the response from the server
	response := make([]byte, 1)
	io.ReadFull(h.ControlStream, response)
	switch response[0] {
	case P.ReturnCodeAccepted:
		return nil
	case P.ReturnCodeUnrecognizedToken:
		return E.New("unrecognized token")
	case P.ReturnCodePortInUse:
		return E.New("port in use")
	case P.ReturnCodeOtherError:
		return E.New("other error")
	default:
		return E.New("unknown response code")
	}
}
