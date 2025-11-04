package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/andreweacott/tado-prometheus-exporter/pkg/collector"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/config"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/logger"
	"github.com/andreweacott/tado-prometheus-exporter/pkg/metrics"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartServer starts the HTTP server with Prometheus endpoints
func StartServer(
	ctx context.Context,
	cfg *config.Config,
	tadoCollector *collector.TadoCollector,
	metricDescriptors *metrics.MetricDescriptors,
	log *logger.Logger,
	exporterMetrics *metrics.ExporterMetrics,
) error {
	// Create a custom registry for our metrics
	registry := prometheus.NewRegistry()

	// Register the Tado collector
	// The collector includes both Tado metrics and exporter health metrics (if provided)
	if err := registry.Register(tadoCollector); err != nil {
		return fmt.Errorf("failed to register Tado collector: %w", err)
	}

	// Note: ExporterMetrics are already registered with the default registry by NewExporterMetrics()
	// and are collected through the TadoCollector's Collect() method

	// Create HTTP server
	mux := http.NewServeMux()

	// Register /metrics endpoint with our custom registry
	metricsHandler := promhttp.HandlerFor(registry, promhttp.HandlerOpts{
		EnableOpenMetrics: true,
		Timeout:           time.Duration(cfg.ScrapeTimeout) * time.Second,
	})
	mux.Handle("/metrics", metricsHandler)

	// Register /health endpoint
	mux.HandleFunc("/health", handleHealth)

	// Create HTTP server
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Port),
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  65 * time.Second,
	}

	// Start server in background
	serverErrors := make(chan error, 1)
	go func() {
		log.Info("Starting HTTP server", "address", server.Addr, "port", cfg.Port)
		log.Info("Metrics endpoint available", "url", fmt.Sprintf("http://localhost:%d/metrics", cfg.Port))
		log.Info("Health endpoint available", "url", fmt.Sprintf("http://localhost:%d/health", cfg.Port))
		serverErrors <- server.ListenAndServe()
	}()

	// Wait for context cancellation or server error
	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			return fmt.Errorf("HTTP server error: %w", err)
		}
		return nil

	case <-ctx.Done():
		// Graceful shutdown
		log.Info("Shutting down HTTP server...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := server.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("HTTP server shutdown error: %w", err)
		}

		log.Info("HTTP server stopped")
		return nil
	}
}

// handleHealth handles the /health endpoint
func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`{"status":"ok"}`))
}

// SetupGracefulShutdown sets up signal handlers for graceful shutdown
// Returns a context that is cancelled on interrupt or termination signal
func SetupGracefulShutdown() context.Context {
	ctx, cancel := context.WithCancel(context.Background())

	// Handle OS signals
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		fmt.Printf("Received signal: %v\n", sig)
		cancel()
	}()

	return ctx
}
