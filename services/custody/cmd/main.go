// Package main is the entrypoint for the custody REST API server.
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/p2p/custody/v2/api"
	"github.com/p2p/custody/v2/internal/client"
	"github.com/p2p/custody/v2/internal/service"
)

const (
	defaultPort                 = "8080"
	defaultBroadcasterEndpoint = "localhost:50067"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	broadcasterEndpoint := os.Getenv("BROADCASTER_ENDPOINT")
	if broadcasterEndpoint == "" {
		broadcasterEndpoint = defaultBroadcasterEndpoint
	}

	// --- Wire dependencies ---
	broadcasterClient, err := client.NewBroadcasterClient(broadcasterEndpoint)
	if err != nil {
		logger.Fatal("failed to create broadcaster client", zap.Error(err))
	}
	defer broadcasterClient.Close()

	custodyService := service.NewCustodyService(logger, broadcasterClient)
	server := api.NewServer(logger, custodyService)

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	addr := ":" + port
	httpServer := &http.Server{Addr: addr, Handler: mux}

	// --- Start server in background ---
	errChan := make(chan error, 1)
	go func() {
		logger.Info("custody API server starting", zap.String("addr", addr))
		if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errChan <- err
		}
	}()

	// --- Graceful shutdown on SIGINT / SIGTERM ---
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Fatal("server error", zap.Error(err))
	case sig := <-sigChan:
		logger.Info("shutting down", zap.String("signal", sig.String()))
		httpServer.Close()
	}
}
