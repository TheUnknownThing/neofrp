package parser

import (
	"encoding/json"
	"fmt"
	"os"

	"neofrp/common/config"
)

// ParseClientConfig reads and parses the client configuration file.
func ParseClientConfig(filePath string) (*config.ClientConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	clientConfig := &config.ClientConfig{}
	// Set default values before decoding
	clientConfig.LogConfig.LogLevel = "info"

	if err := decoder.Decode(clientConfig); err != nil {
		return nil, err
	}

	return clientConfig, nil
}

// ParseServerConfig reads and parses the server configuration file.
func ParseServerConfig(filePath string) (*config.ServerConfig, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	serverConfig := &config.ServerConfig{}
	// Set default values before decoding
	serverConfig.LogConfig.LogLevel = "info"

	if err := decoder.Decode(serverConfig); err != nil {
		return nil, err
	}

	return serverConfig, nil
}

func ValidateClientConfig(config *config.ClientConfig) error {
	if config.Token == "" {
		return fmt.Errorf("token is required")
	}
	if config.TransportConfig.Protocol != "quic" && config.TransportConfig.Protocol != "tcp" {
		return fmt.Errorf("invalid transport protocol: %s. Options are: quic, tcp", config.TransportConfig.Protocol)
	}
	if config.TransportConfig.IP == "" {
		return fmt.Errorf("server IP is required in field 'transport'")
	}
	if config.TransportConfig.Port == 0 {
		return fmt.Errorf("server port is required in field 'transport'")
	}
	for i, conn := range config.ConnectionConfigs {
		if conn.Type != "tcp" && conn.Type != "udp" {
			return fmt.Errorf("connection %d: invalid connection type: %s", i, conn.Type)
		}
		if conn.ServerPort == 0 {
			return fmt.Errorf("connection %d: server_port is required", i)
		}
	}
	return nil
}

func ValidateServerConfig(config *config.ServerConfig) error {
	if config.TransportConfig.Protocol != "quic" && config.TransportConfig.Protocol != "tcp" {
		return fmt.Errorf("invalid transport protocol: %s. Options are: quic, tcp", config.TransportConfig.Protocol)
	}
	if config.TransportConfig.Port == 0 {
		return fmt.Errorf("server port is required in field 'transport'")
	}
	if len(config.RecognizedTokens) == 0 {
		return fmt.Errorf("at least one recognized token is required for the server to start")
	}
	return nil
}
