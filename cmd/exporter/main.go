package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/andreweacott/tado-prometheus-exporter/pkg/auth"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/collector"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/config"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/metrics"
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

	// Create context with graceful shutdown support
	ctx := SetupGracefulShutdown()

	// Phase 2: Initialize OAuth authentication
	tadoClient, metricDescs, err := initializeAuth(context.Background(), cfg)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Authentication error: %v\n", err)
		os.Exit(1)
	}

	// Phase 3: Initialize Prometheus metrics and HTTP server
	if err := initializeMetricsAndServer(ctx, cfg, tadoClient, metricDescs); err != nil {
		fmt.Fprintf(os.Stderr, "Server initialization error: %v\n", err)
		os.Exit(1)
	}
}

// initializeAuth handles OAuth authentication and returns authenticated Tado client and metrics descriptors
func initializeAuth(ctx context.Context, cfg *config.Config) (*collector.TadoCollector, *metrics.MetricDescriptors, error) {
	// Create metric descriptors first (before authentication, so we can fail fast)
	metricDescs, err := metrics.NewMetricDescriptors()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metric descriptors: %w", err)
	}

	// Create authenticated Tado client with encrypted token storage
	// This handles:
	// - Loading existing token if valid
	// - Performing device code OAuth flow if no valid token
	// - Storing encrypted token with passphrase
	fmt.Println("Initializing Tado authentication...")
	tadoClient, err := auth.NewAuthenticatedTadoClient(ctx, cfg.TokenPath, cfg.TokenPassphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("authentication failed: %w", err)
	}

	fmt.Printf("Successfully authenticated. Token stored at: %s (encrypted with passphrase)\n", cfg.TokenPath)

	scrapeTimeout := time.Duration(cfg.ScrapeTimeout) * time.Second
	tadoCollector := collector.NewTadoCollector(tadoClient, metricDescs, scrapeTimeout, cfg.HomeID)

	return tadoCollector, metricDescs, nil
}

// initializeMetricsAndServer initializes metrics and starts the HTTP server
func initializeMetricsAndServer(ctx context.Context, cfg *config.Config, tadoCollector *collector.TadoCollector, metricDescs *metrics.MetricDescriptors) error {
	fmt.Println("Prometheus metrics registered successfully")

	// Start HTTP server with graceful shutdown
	return StartServer(ctx, cfg, tadoCollector, metricDescs)
}
