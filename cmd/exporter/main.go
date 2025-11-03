package main

import (
	"fmt"
	"os"

	"github.com/andreweacott/tado-prometheus-exporter/pkg/config"
)

func main() {
	// Load configuration
	cfg := config.Load()

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("tado-prometheus-exporter starting with config: %s\n", cfg)

	// TODO: Phase 2 - Implement OAuth authentication
	// TODO: Phase 3 - Initialize Prometheus collector and HTTP server
	// TODO: Phase 4 - Implement metrics collection logic
}
