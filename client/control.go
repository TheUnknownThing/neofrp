package client

import (
	E "errors"
	"fmt"
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

func (h *ControlHandler) Negotiate() error {
	// Build the port configuration data
	// [LENGTH] [DATA] where DATA is []struct { [TYPE] [PORT] }
	var portData []byte

	// Add TCP ports
	for _, connConfig := range h.Config.ConnectionConfigs {
		if connConfig.Type == "tcp" {
			portData = append(portData, C.PortTypeTCP)
			portData = append(portData, byte(connConfig.ServerPort>>8))
			portData = append(portData, byte(connConfig.ServerPort&0xFF))
		}
	}

	// Add UDP ports
	for _, connConfig := range h.Config.ConnectionConfigs {
		if connConfig.Type == "udp" {
			portData = append(portData, C.PortTypeUDP)
			portData = append(portData, byte(connConfig.ServerPort>>8))
			portData = append(portData, byte(connConfig.ServerPort&0xFF))
		}
	}

	// Send the port configuration
	length := len(portData) / 3 // Each port entry is 3 bytes
	msg := append([]byte{byte(length)}, portData...)
	_, err := h.ControlStream.Write(msg)
	if err != nil {
		return fmt.Errorf("failed to send port configuration: %v", err)
	}

	// Read the response
	// [LENGTH] [RESPONSE]
	lengthBuf := make([]byte, 1)
	_, err = io.ReadFull(h.ControlStream, lengthBuf)
	if err != nil {
		return fmt.Errorf("failed to read response length: %v", err)
	}

	responseLength := int(lengthBuf[0])
	if responseLength == 0 {
		return nil // No ports configured
	}

	responses := make([]byte, responseLength)
	_, err = io.ReadFull(h.ControlStream, responses)
	if err != nil {
		return fmt.Errorf("failed to read port responses: %v", err)
	}

	// Check responses
	portIndex := 0
	for _, connConfig := range h.Config.ConnectionConfigs {
		if portIndex >= len(responses) {
			break
		}

		response := responses[portIndex]
		switch response {
		case P.ReturnCodeAccepted:
			// Port accepted
		case P.ReturnCodePortInUse:
			return fmt.Errorf("port %d (%s) is in use", connConfig.ServerPort, connConfig.Type)
		case P.ReturnCodeOtherError:
			return fmt.Errorf("server error for port %d (%s)", connConfig.ServerPort, connConfig.Type)
		default:
			return fmt.Errorf("unknown response code %d for port %d (%s)", response, connConfig.ServerPort, connConfig.Type)
		}
		portIndex++
	}

	return nil
}
