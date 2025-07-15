package client

import (
	"github.com/charmbracelet/log"
	"internal.github/Norb/frp/common/config"
)

func Run(config *config.ClientConfig) {
	// Initialize the client service with the provided configuration
	log.Infof("Starting client with config: %+v", config)
}