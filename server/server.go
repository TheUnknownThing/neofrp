package server

import (
	"github.com/charmbracelet/log"
	"internal.github/Norb/frp/common/config"
)

// Run initializes the server service with the provided configuration
func Run(config *config.ServerConfig) {
	// Here you would typically set up the server, start listening for connections, etc.
	log.Infof("Starting server with config: %+v", config)

	// Example: Initialize server components, start listening, etc.
	// This is a placeholder for actual server logic.
	// In a real implementation, you would handle incoming connections,
	// manage sessions, etc.
}
