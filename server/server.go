package server

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"math/big"
	"neofrp/common/config"
	"neofrp/common/multidialer"
	"net"
	"time"

	"github.com/charmbracelet/log"
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
}

func GetTLSConfig() (*tls.Config, error) {
	// Generate a self-signed certificate for the server
	cert, err := generateSelfSignedCert()
	if err != nil {
		return nil, fmt.Errorf("failed to generate self-signed certificate: %v", err)
	}

	return &tls.Config{
		Certificates:       []tls.Certificate{cert},
		InsecureSkipVerify: true,
		ServerName:         "",             // Empty server name to avoid SNI issues
		NextProtos:         []string{"h3"}, // HTTP/3 for QUIC
		MinVersion:         tls.VersionTLS12,
	}, nil
}

func generateSelfSignedCert() (tls.Certificate, error) {
	// Generate a private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create certificate template
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject: pkix.Name{
			Organization:  []string{"FRP"},
			Country:       []string{"US"},
			Province:      []string{""},
			Locality:      []string{""},
			StreetAddress: []string{""},
			PostalCode:    []string{""},
		},
		NotBefore:             time.Now(),
		NotAfter:              time.Now().Add(365 * 24 * time.Hour), // Valid for 1 year
		KeyUsage:              x509.KeyUsageKeyEncipherment | x509.KeyUsageDigitalSignature,
		ExtKeyUsage:           []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		BasicConstraintsValid: true,
		IPAddresses:           []net.IP{net.IPv4(127, 0, 0, 1), net.IPv6loopback}, // localhost
		DNSNames:              []string{"localhost"},
	}

	// Create certificate
	certDER, err := x509.CreateCertificate(rand.Reader, &template, &template, &privateKey.PublicKey, privateKey)
	if err != nil {
		return tls.Certificate{}, err
	}

	// Create tls.Certificate
	cert := tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  privateKey,
	}

	return cert, nil
}
