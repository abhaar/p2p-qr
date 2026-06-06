package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/p2p/blockchain/evm/protocol/v2/internal/domain"
	"github.com/p2p/blockchain/evm/protocol/v2/internal/service"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Blockchain service starting up...")

	conf, err := service.NewConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	blockchainClient, err := domain.NewEVMClient(conf.BlockchainNodeEndpoint)
	if err != nil {
		logger.Fatal("failed to create blockchain client", zap.Error(err))
	}

	blockchainServer := service.NewBlockchainService(blockchainClient, *conf)

	lis, err := net.Listen("tcp", conf.GRPCServicePort)
	if err != nil {
		logger.Fatal("failed to bind grpc server", zap.Error(err))
	}

	grpcServer := grpc.NewServer()
	protocol.RegisterProtocolServiceServer(grpcServer, blockchainServer)
	reflection.Register(grpcServer)

	errChan := make(chan error, 1)

	go func() {
		logger.Info("gRPC server started listening", zap.String("port", conf.GRPCServicePort))
		if err := grpcServer.Serve(lis); err != nil {
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
		logger.Info("Stopping gRPC server gracefully...")
		grpcServer.GracefulStop()
		logger.Info("Shutdown complete.")
	}
}
