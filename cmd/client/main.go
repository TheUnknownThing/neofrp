package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/charmbracelet/log"
	"internal.github/Norb/frp/client"
	"internal.github/Norb/frp/common/parser"
)

func main() {
	var configFile string
	flag.StringVar(&configFile, "c", "", "Client configuration file")
	flag.Parse()

	if configFile == "" {
		fmt.Println("Usage: frpc -c <config_file>")
		os.Exit(1)
	}

	config, err := parser.ParseClientConfig(configFile)
	if err != nil {
		log.Errorf("Failed to parse config file: %v", err)
		os.Exit(1)
	}

	if err := parser.ValidateClientConfig(config); err != nil {
		log.Errorf("Invalid client config: %v\n", err)
		os.Exit(1)
	}

	log.Infof("Parsed client config")
	
	// Run the client service
	client.Run(config)
}
