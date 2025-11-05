package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/andreweacott/tado-prometheus-exporter/pkg/auth"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/collector"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/config"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/logger"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/metrics"
)

func main() {
	cfg := config.Load()

	if err := cfg.Validate(); err != nil {
		fmt.Fprintf(os.Stderr, "Configuration error: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.New(cfg.LogLevel, "text")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Logger initialization error: %v\n", err)
		os.Exit(1)
	}

	log.Info("tado-prometheus-exporter starting", "config", cfg.String())

	ctx := SetupGracefulShutdown()

	tadoClient, metricDescs, err := initializeAuth(context.Background(), cfg, log)
	if err != nil {
		log.Error("Authentication failed", "error", err.Error())
		os.Exit(1)
	}

	exporterMetrics, err := metrics.NewExporterMetrics()
	if err != nil {
		log.Error("Exporter metrics initialization failed", "error", err.Error())
		os.Exit(1)
	}
	log.Info("Exporter health metrics initialized")

	if err := initializeMetricsAndServer(ctx, cfg, tadoClient, metricDescs, exporterMetrics, log); err != nil {
		log.Error("Server initialization failed", "error", err.Error())
		os.Exit(1)
	}
}

// initializeAuth handles OAuth authentication and returns authenticated Tado client and metrics descriptors
func initializeAuth(ctx context.Context, cfg *config.Config, log *logger.Logger) (*collector.TadoCollector, *metrics.MetricDescriptors, error) {
	metricDescs, err := metrics.NewMetricDescriptors()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create metric descriptors: %w", err)
	}

	// Create authenticated Tado client with encrypted token storage
	// This handles:
	// - Loading existing token if valid
	// - Performing device code OAuth flow if no valid token
	// - Storing encrypted token with passphrase
	log.Info("Initializing Tado authentication...")
	tadoClientRaw, err := auth.NewAuthenticatedTadoClient(ctx, cfg.TokenPath, cfg.TokenPassphrase)
	if err != nil {
		return nil, nil, fmt.Errorf("authentication failed: %w", err)
	}

	log.Info("Successfully authenticated", "token_path", cfg.TokenPath)

	tadoClient := collector.NewTadoClientAdapter(tadoClientRaw)

	scrapeTimeout := time.Duration(cfg.ScrapeTimeout) * time.Second
	tadoCollector := collector.NewTadoCollectorWithLogger(tadoClient, metricDescs, scrapeTimeout, cfg.HomeID, log)

	return tadoCollector, metricDescs, nil
}

// initializeMetricsAndServer initializes metrics and starts the HTTP server
func initializeMetricsAndServer(ctx context.Context, cfg *config.Config, tadoCollector *collector.TadoCollector, metricDescs *metrics.MetricDescriptors, exporterMetrics *metrics.ExporterMetrics, log *logger.Logger) error {
	tadoCollector.WithExporterMetrics(exporterMetrics)

	log.Info("Prometheus metrics registered successfully")

	return StartServer(ctx, cfg, tadoCollector, metricDescs, log, exporterMetrics)
}
