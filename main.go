package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"mm-mcp-server/internal/config"
	"mm-mcp-server/internal/gateway"
	"mm-mcp-server/internal/logging"
	"mm-mcp-server/internal/mcp"
	"mm-mcp-server/internal/tools"
)

func main() {
	cfg := config.Load()

	logger := logging.Setup(cfg.LogLevel, cfg.LogFormat)

	if cfg.APIBase == "" {
		logger.Error("API_BASE is required")
		os.Exit(1)
	}

	registry := tools.NewRegistry()
	gw := gateway.New(cfg.APIBase)
	server := mcp.New(registry, gw)

	handler := logging.RequestLogger(server.Handler())

	srv := &http.Server{
		Addr:              fmt.Sprintf(":%s", cfg.Port),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      90 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go func() {
		logger.Info("mcp server listening", "address", srv.Addr, "tools", len(registry.All()), "api_base", cfg.APIBase)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			logger.Error("server listen failed", "error", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	logger.Info("shutting down gracefully")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("error during shutdown", "error", err)
	} else {
		logger.Info("mcp server stopped cleanly")
	}
}
