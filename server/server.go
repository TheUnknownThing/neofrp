package server

import (
	"context"
	"fmt"
	"net"

	"github.com/charmbracelet/log"
	"neofrp/common/config"
	C "neofrp/common/constant"
	"neofrp/common/multidialer"
	"neofrp/common/safemap"
)

// Run initializes the server service with the provided configuration
func Run(config *config.ServerConfig) {
	log.Infof("Starting server with config: %+v", config)
	tlsConfig, err := GetTLSConfig()
	if err != nil {
		log.Errorf("Failed to get TLS config: %v", err)
		return
	}
	ctx := context.Background()
	// Register the portmap into ctx
	ctx = context.WithValue(ctx, C.ContextPortMapKey, safemap.NewSafeMap[C.TaggedPort, C.SessionIndexCompound]())
	// Setup all the ports

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
	// Open a control stream for the session
	controlConn, err := session.AcceptStream(ctx)
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
	err = controlHandler.Negotiate(config)
	if err != nil {
		log.Errorf("Negotiation failed: %v", err)
		return
	}
}

func SetupPorts(ctx context.Context, config *config.ServerConfig) error {
	portMap := ctx.Value("portmap").(*safemap.SafeMap[C.TaggedPort, C.SessionIndexCompound])
	if portMap == nil {
		return fmt.Errorf("portmap not found in context")
	}

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
		log.Infof("Registered port %d with type %d", port, C.PortTypeTCP)
	}
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
		log.Infof("Registered port %d with type %d", port, C.PortTypeUDP)
	}

	return nil
}
