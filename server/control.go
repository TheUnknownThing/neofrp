package server

import (
	"context"
	"fmt"
	"io"
	"os"
	"slices"
	"time"

	"neofrp/common/config"
	C "neofrp/common/constant"
	"neofrp/common/multidialer"
	P "neofrp/common/protocol"

	"github.com/charmbracelet/log"
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

func CancelConnection(ctx context.Context) {
	// Prefer using a stored session cancel func if available
	if v := ctx.Value(C.ContextSessionCancelKey); v != nil {
		if cancelFunc, ok := v.(context.CancelFunc); ok && cancelFunc != nil {
			log.Debug("Calling session cancel function")
			cancelFunc()
		}
	}
	// Backwards compatibility: attempt to send signal if channel present
	if v := ctx.Value(C.ContextSignalChanKey); v != nil {
		if cancelChan, ok := v.(chan os.Signal); ok && cancelChan != nil {
			log.Debug("Sending interrupt signal to cancel connection via signal channel")
			select {
			case cancelChan <- os.Interrupt:
			default:
				log.Debug("Cancel channel is full or closed, skipping signal")
			}
		}
	}
}

func RunControlLoop(ctx context.Context, controlConn multidialer.Stream, session multidialer.Session, config *config.ServerConfig) {
	defer controlConn.Close()

	// Send periodic keep-alive messages to maintain the connection
	keepAliveTicker := time.NewTicker(C.KeepAliveInterval)
	defer keepAliveTicker.Stop()

	// Channel to handle control messages
	controlMsg := make(chan []byte, 10)

	// Start a goroutine to read control messages
	go func() {
		buf := make([]byte, 1024)
		for {
			select {
			case <-ctx.Done():
				return
			default:
				n, err := controlConn.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Errorf("Error reading from control connection: %v", err)
					}
					return
				}

				if n > 0 {
					// Copy the data to avoid race conditions
					msg := make([]byte, n)
					copy(msg, buf[:n])
					select {
					case controlMsg <- msg:
					case <-ctx.Done():
						return
					}
				}
			}
		}
	}()

	// Setup keepalive monitoring
	go func() {
		ticker := time.NewTicker(C.KeepAliveTimeout)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				lastKeepAlive := ctx.Value(C.ContextLastKeepAliveKey)
				if lastKeepAlive == nil || time.Since(lastKeepAlive.(time.Time)) > C.KeepAliveTimeout {
					log.Warnf("No keep-alive received in the last %v, closing connection", C.KeepAliveTimeout)
					// Close the connection to the server
					CancelConnection(ctx)
					return
				}
			}
		}
	}()

	// Main control loop
	for {
		select {
		case <-ctx.Done():
			log.Infof("Context cancelled, stopping control connection handler")
			return

		case msg := <-controlMsg:
			log.Debugf("Received control message: %v", msg)
			switch msg[0] {
			case P.ActionKeepAlive:
				// Update last recorded keep-alive time
				ctx = context.WithValue(ctx, C.ContextLastKeepAliveKey, time.Now())
			case P.ActionClose:
				log.Infof("Received active close action from client, closing connection")
				CancelConnection(ctx)
			}

		case <-keepAliveTicker.C:
			// Send keep-alive message
			_, err := controlConn.Write([]byte{P.ActionKeepAlive}) // Simple keep-alive byte
			if err != nil {
				log.Errorf("Failed to send keep-alive: %v", err)
				return
			}
			log.Debugf("Sent keep-alive message")
		}
	}
}
