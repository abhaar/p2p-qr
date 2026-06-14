// Package main is the entrypoint for the custody REST API server.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/p2p/custody/v2/api"
	"github.com/p2p/custody/v2/internal/client"
	"github.com/p2p/custody/v2/internal/consumer"
	"github.com/p2p/custody/v2/internal/service"
)

const (
	defaultPort                   = "8080"
	defaultBroadcasterEndpoint    = "localhost:50067"
	defaultAddressServiceEndpoint = "localhost:50064"
	defaultNatsURL                = "nats://localhost:4222"
	defaultNatsSubject            = "blockchain.events"
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

	addressServiceEndpoint := os.Getenv("ADDRESS_SERVICE_ENDPOINT")
	if addressServiceEndpoint == "" {
		addressServiceEndpoint = defaultAddressServiceEndpoint
	}

	natsURL := os.Getenv("NATS_URL")
	if natsURL == "" {
		natsURL = defaultNatsURL
	}

	natsSubject := os.Getenv("NATS_SUBJECT")
	if natsSubject == "" {
		natsSubject = defaultNatsSubject
	}

	// --- Wire dependencies ---
	broadcasterClient, err := client.NewBroadcasterClient(broadcasterEndpoint)
	if err != nil {
		logger.Fatal("failed to create broadcaster client", zap.Error(err))
	}
	defer broadcasterClient.Close()

	addressClient, err := client.NewAddressClient(addressServiceEndpoint)
	if err != nil {
		logger.Fatal("failed to create address client", zap.Error(err))
	}
	defer addressClient.Close()

	custodyService := service.NewCustodyService(logger, broadcasterClient, addressClient)

	// --- Start event consumer in background ---
	eventConsumer, err := consumer.NewEventConsumer(logger, addressClient, natsURL, natsSubject, custodyService.Repo())
	if err != nil {
		logger.Fatal("failed to initialize NATS event consumer", zap.Error(err))
	}
	defer eventConsumer.Close()

	consumerCtx, cancelConsumer := context.WithCancel(context.Background())
	defer cancelConsumer()

	go func() {
		if err := eventConsumer.Start(consumerCtx); err != nil {
			logger.Error("event consumer stopped with error", zap.Error(err))
		}
	}()

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
		cancelConsumer() // Stop NATS consumer first
		httpServer.Close()
	}
}
