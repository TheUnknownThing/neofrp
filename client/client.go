package client

import (
	"context"
	"crypto/tls"
	"fmt"
	"neofrp/common/multidialer"
	"net"

	"neofrp/common/config"

	"github.com/charmbracelet/log"
)

func Run(config *config.ClientConfig) {
	// Initialize the client service with the provided configuration
	log.Infof("Starting client with config: %+v", config)
	// First create the master connection to the server
	tlsConfig, err := GetTLSConfig()
	if err != nil {
		log.Errorf("Failed to get TLS config: %v", err)
		return
	}
	session, err := multidialer.Dial(
		context.Background(),
		config.TransportConfig.Protocol,
		net.JoinHostPort(config.TransportConfig.IP, fmt.Sprintf("%d", config.TransportConfig.Port)),
		tlsConfig,
	)
	if err != nil {
		log.Errorf("Failed to connect to server: %v", err)
		return
	}
	log.Infof("Connected to server at %s", session.RemoteAddr())

	// Build the control connection
	ctx := context.Background()
	controlConn, err := session.OpenStream(ctx)
	if err != nil {
		log.Errorf("Failed to open control stream: %v", err)
		return
	}

	controlHandler := NewControlHandler(config, controlConn)
	err = controlHandler.Handshake()
	if err != nil {
		log.Errorf("Handshake failed: %v", err)
		return
	}
}

func GetTLSConfig() (*tls.Config, error) {
	return &tls.Config{
		InsecureSkipVerify: true,
		ServerName:         "",             // Empty server name to avoid SNI issues
		NextProtos:         []string{"h3"}, // HTTP/3 for QUIC
		MinVersion:         tls.VersionTLS12,
	}, nil
}
