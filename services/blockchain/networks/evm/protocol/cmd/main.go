package main

import (
	"context"
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
)

func main() {
	_, cancel := context.WithCancel(context.Background())
	addShutdownHook(cancel)

	log.Println("Blockchain service starting up...")

	logger, err := zap.NewProduction()
	if err != nil {
		logger.Fatal("failed to initialize zap logger")
	}
	defer logger.Sync()

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
	if err = grpcServer.Serve(lis); err != nil {
		logger.Fatal("failed to serve", zap.Error(err))
	}
	logger.Info("gRPC server started listening", zap.String("port", conf.GRPCServicePort))
	defer grpcServer.GracefulStop()
}

func addShutdownHook(cancelFn context.CancelFunc) {
	// Graceful shutdown
	c := make(chan os.Signal, 1)
	signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Received '%s' signal. Shutting down...\n", sig)
		cancelFn()
	}()
}
