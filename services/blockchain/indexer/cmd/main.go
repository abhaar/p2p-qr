package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/p2p/blockchain/indexer/v2/internal/domain"
	"github.com/p2p/blockchain/indexer/v2/internal/publisher"
	"github.com/p2p/blockchain/indexer/v2/internal/repository"
	"github.com/p2p/blockchain/indexer/v2/internal/service"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Blockchain indexer starting up...")

	conf, err := service.NewConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	protocolConn, err := grpc.NewClient(conf.ProtocolServiceEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("failed to connect to prtocol service", zap.Error(err))
	}
	defer protocolConn.Close()

	protocolClient := protocol.NewProtocolServiceClient(protocolConn)
	repo := repository.NewInMemoryRepo(domain.BlockHeight(conf.StartingBlock))
	producer := &publisher.NoopPublisher{}

	indexer := service.New(logger, protocolClient, repo, producer)
	errChan := make(chan error, 1)

	go func() {
		logger.Info("starting indexer")
		if err := indexer.Run(ctx); err != nil {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		logger.Fatal("failed to serve gRPC server", zap.Error(err))
	case sig := <-sigChan:
		logger.Info("Received shutdown signal", zap.String("signal", sig.String()))
		cancel()
		logger.Info("Shutdown complete.")
	}
}
