package server

import (
	"neofrp/common/config"

	"github.com/charmbracelet/log"
)

// Run initializes the server service with the provided configuration
func Run(config *config.ServerConfig) {
	log.Infof("Starting server with config: %+v", config)

}
