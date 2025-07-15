package main

import (
	"flag"
	"fmt"
	"os"

	"neofrp/common/parser"

	"neofrp/server"

	"github.com/charmbracelet/log"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "", "Server configuration file")
	flag.Parse()

	if configFile == "" {
		fmt.Println("Usage: frps -c <config_file>")
		os.Exit(1)
	}

	config, err := parser.ParseServerConfig(configFile)
	if err != nil {
		log.Errorf("Failed to parse config file: %v", err)
		os.Exit(1)
	}

	if err := parser.ValidateServerConfig(config); err != nil {
		log.Errorf("Invalid server config: %v", err)
		os.Exit(1)
	}

	log.Info("Parsed server config")

	// Run the server service
	server.Run(config)
}
