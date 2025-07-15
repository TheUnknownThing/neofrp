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

func (h *ControlHandler) Negotiate() error {
	// Send acceptance response for handshake
	_, err := h.conn.Write([]byte{P.ReturnCodeAccepted})
	if err != nil {
		return fmt.Errorf("failed to send handshake response: %v", err)
	}

	// Read the port configuration from client
	// [LENGTH] [DATA]
	lengthBuf := make([]byte, 1)
	_, err = io.ReadFull(h.conn, lengthBuf)
	if err != nil {
		return fmt.Errorf("failed to read port config length: %v", err)
	}

	length := int(lengthBuf[0])
	if length == 0 {
		// No ports to configure
		_, err = h.conn.Write([]byte{0, P.ReturnCodeAccepted})
		return err
	}

	// Read the port data: []struct { [TYPE] [PORT] }
	portData := make([]byte, length*3) // Each port entry is 3 bytes (1 type + 2 port)
	_, err = io.ReadFull(h.conn, portData)
	if err != nil {
		return fmt.Errorf("failed to read port data: %v", err)
	}

	// Parse the port requests and check availability
	responses := make([]byte, length)
	for i := 0; i < length; i++ {
		portType := portData[i*3]
		port := uint16(portData[i*3+1])<<8 | uint16(portData[i*3+2])

		// Check if port is available
		isAvailable := false
		switch portType {
		case C.PortTypeTCP:
			isAvailable = slices.Contains(h.config.ConnectionConfig.TCPPorts, C.PortType(port))
		case C.PortTypeUDP:
			isAvailable = slices.Contains(h.config.ConnectionConfig.UDPPorts, C.PortType(port))
		}

		if isAvailable {
			responses[i] = P.ReturnCodeAccepted
		} else {
			responses[i] = P.ReturnCodePortInUse
		}
	}

	// Send response: [LENGTH] [RESPONSE]
	response := append([]byte{byte(length)}, responses...)
	_, err = h.conn.Write(response)
	if err != nil {
		return fmt.Errorf("failed to send port responses: %v", err)
	}

	return nil
}
