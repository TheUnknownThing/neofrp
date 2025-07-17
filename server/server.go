package server

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/signal"
	"syscall"

	"neofrp/common/config"
	C "neofrp/common/constant"
	"neofrp/common/multidialer"
	"neofrp/common/safemap"

	"github.com/charmbracelet/log"
)

// Run initializes the server service with the provided configuration
func Run(config *config.ServerConfig) {
	log.Debugf("Run using config: %+v", config)
	tlsConfig, err := GetTLSConfig()
	if err != nil {
		log.Errorf("Failed to get TLS config: %v", err)
		return
	}
	ctx := context.Background()
	// Register the portmap into ctx
	ctx = context.WithValue(ctx, C.ContextPortMapKey, safemap.NewSafeMap[C.TaggedPort, C.SessionIndexCompound]())

	// Setup all the ports
	err = SetupPorts(ctx, config)
	if err != nil {
		log.Errorf("Failed to setup ports: %v", err)
		return
	}

	// Start the server with the TLS configuration
	listener, err := multidialer.Listen(
		ctx,
		config.TransportConfig.Protocol,
		net.JoinHostPort("0.0.0.0", fmt.Sprintf("%d", config.TransportConfig.Port)),
		tlsConfig,
	)
	if err != nil {
		log.Errorf("Failed to start listener: %v", err)
		return
	}
	log.Infof("Server listening on %s", listener.Addr())

	// Accept sessions from clients
	for {
		session, err := listener.AcceptSession()
		if err != nil {
			log.Errorf("Failed to accept session: %v", err)
			continue
		}
		log.Infof("Accepted connection from %s", session.RemoteAddr())
		go handleSession(ctx, config, session)
	}
}

func handleSession(ctx context.Context, config *config.ServerConfig, session multidialer.Session) {
	// Create a cancellable context for this session
	sessionCtx, sessionCancel := context.WithCancel(ctx)
	defer sessionCancel()

	// Open a signal channel to handle cancellation
	cancelChan := make(chan os.Signal, 1)
	signal.Notify(cancelChan, os.Interrupt, syscall.SIGTERM)
	sessionCtx = context.WithValue(sessionCtx, C.ContextSignalChanKey, cancelChan)

	// Cleanup function to ensure resources are properly released
	defer func() {
		signal.Stop(cancelChan)
		close(cancelChan)
		log.Infof("Session cleanup completed for %s", session.RemoteAddr())
	}()

	// Open a control stream for the session
	controlConn, err := session.AcceptStream(sessionCtx)
	if err != nil {
		log.Errorf("Failed to accept control stream: %v", err)
		return
	}
	defer controlConn.Close()

	// Read the client handshake
	controlHandler := NewControlHandler(config, controlConn)
	err = controlHandler.Handshake()
	if err != nil {
		log.Errorf("Handshake failed: %v", err)
		return
	}
	log.Infof("Handshake successful with client %s", session.RemoteAddr())

	// Register this session in the portmap for all configured ports
	portMap := ctx.Value(C.ContextPortMapKey).(*safemap.SafeMap[C.TaggedPort, C.SessionIndexCompound])
	if portMap != nil {
		// Register session for TCP ports
		for _, port := range config.ConnectionConfig.TCPPorts {
			taggedPort := C.TaggedPort{
				PortType: "tcp",
				Port:     C.PortType(port),
			}
			sessionIndex := C.SessionIndexCompound{
				Session: &session,
				Index:   0,
			}
			portMap.Set(taggedPort, sessionIndex)
			log.Infof("Registered session %s for TCP port %d", session.RemoteAddr(), port)
		}

		// Register session for UDP ports
		for _, port := range config.ConnectionConfig.UDPPorts {
			taggedPort := C.TaggedPort{
				PortType: "udp",
				Port:     C.PortType(port),
			}
			sessionIndex := C.SessionIndexCompound{
				Session: &session,
				Index:   0,
			}
			portMap.Set(taggedPort, sessionIndex)
			log.Infof("Registered session %s for UDP port %d", session.RemoteAddr(), port)
		}
	}

	err = controlHandler.Negotiate()
	if err != nil {
		log.Errorf("Negotiation failed: %v", err)
		return
	}

	// Setup the control feedback loop
	go RunControlLoop(sessionCtx, controlConn, session, config)

	// Wait for cancellation signal
	<-cancelChan
	log.Infof("Received cancellation signal, closing session %s", session.RemoteAddr())

	// Clean up session from portmap
	if portMap != nil {
		// Remove session from TCP ports
		for _, port := range config.ConnectionConfig.TCPPorts {
			taggedPort := C.TaggedPort{
				PortType: "tcp",
				Port:     C.PortType(port),
			}
			portMap.Delete(taggedPort)
			log.Infof("Unregistered session %s from TCP port %d", session.RemoteAddr(), port)
		}

		// Remove session from UDP ports
		for _, port := range config.ConnectionConfig.UDPPorts {
			taggedPort := C.TaggedPort{
				PortType: "udp",
				Port:     C.PortType(port),
			}
			portMap.Delete(taggedPort)
			log.Infof("Unregistered session %s from UDP port %d", session.RemoteAddr(), port)
		}
	}

	// Close the session gracefully
	err = session.Close(fmt.Errorf("session closed by server due to cancellation"))
	if err != nil {
		log.Errorf("Failed to close session: %v", err)
	} else {
		log.Infof("Session %s closed successfully", session.RemoteAddr())
	}
}

func SetupPorts(ctx context.Context, config *config.ServerConfig) error {
	portMap := ctx.Value(C.ContextPortMapKey).(*safemap.SafeMap[C.TaggedPort, C.SessionIndexCompound])
	if portMap == nil {
		return fmt.Errorf("portmap not found in context")
	}

	// Setup TCP ports
	for _, port := range config.ConnectionConfig.TCPPorts {
		taggedPort := C.TaggedPort{
			PortType: "tcp",
			Port:     C.PortType(port),
		}
		sessionIndex := C.SessionIndexCompound{
			Session: nil, // This will be set when a session is accepted
			Index:   0,   // This will be set when a session is accepted
		}
		portMap.Set(taggedPort, sessionIndex)

		// Create and start TCP listener
		tcpListener := &TCPPortListener{
			TaggedPort: taggedPort,
			PortMap:    portMap,
		}
		err := tcpListener.Start()
		if err != nil {
			return fmt.Errorf("failed to start TCP listener for port %d: %v", port, err)
		}
		log.Infof("Started TCP listener for port %d", port)
	}

	// Setup UDP ports
	for _, port := range config.ConnectionConfig.UDPPorts {
		taggedPort := C.TaggedPort{
			PortType: "udp",
			Port:     C.PortType(port),
		}
		sessionIndex := C.SessionIndexCompound{
			Session: nil, // This will be set when a session is accepted
			Index:   0,   // This will be set when a session is accepted
		}
		portMap.Set(taggedPort, sessionIndex)

		// Create and start UDP listener
		udpListener := &UDPPortListener{
			TaggedPort: taggedPort,
			PortMap:    portMap,
			SourceMap:  safemap.NewSafeMap[*net.UDPAddr, *multidialer.Stream](),
		}
		err := udpListener.Start()
		if err != nil {
			return fmt.Errorf("failed to start UDP listener for port %d: %v", port, err)
		}
		log.Infof("Started UDP listener for port %d", port)
	}

	return nil
}
