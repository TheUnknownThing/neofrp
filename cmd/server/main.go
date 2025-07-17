package main

import (
	"flag"
	"fmt"
	"os"

	C "neofrp/common/constant"
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
	logLevel := config.LogConfig.LogLevel
	if logLevel == "" || C.LogLevelMap[logLevel] == 0 {
		log.Warnf("Using default log level INFO")
		log.SetLevel(log.InfoLevel)
	} else {
		log.SetLevel(C.LogLevelMap[logLevel])
	}

	// Run the server service
	server.Run(config)
}
