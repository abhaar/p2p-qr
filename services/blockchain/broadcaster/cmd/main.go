package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/p2p/blockchain/broadcaster/v2/service"
	"github.com/p2p/shared/pb/blockchain/broadcaster"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		log.Fatalf("failed to initialize zap logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("Broadcast service starting up...")

	conf, err := service.NewConfig()
	if err != nil {
		logger.Fatal("failed to load config", zap.Error(err))
	}

	// Listen on default port
	lis, err := net.Listen("tcp", conf.GRPCServicePort)
	if err != nil {
		log.Fatalf("failed to bind grpc server: %v", err)
	}

	protocolConn, err := grpc.NewClient(conf.ProtocolServiceEndpoint, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		logger.Fatal("failed to connect to prtocol service", zap.Error(err))
	}
	defer protocolConn.Close()

	protocolClient := protocol.NewProtocolServiceClient(protocolConn)

	broadcastService := service.NewBroadcastService(logger, conf.NetworkID, protocolClient)

	grpcServer := grpc.NewServer()
	broadcaster.RegisterBroadcastServiceServer(grpcServer, broadcastService)
	reflection.Register(grpcServer)

	errChan := make(chan error, 1)

	go func() {
		log.Printf("gRPC server started listening on port %s\n", conf.GRPCServicePort)
		if err := grpcServer.Serve(lis); err != nil {
			errChan <- err
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-errChan:
		log.Fatalf("failed to serve gRPC server: %v", err)
	case sig := <-sigChan:
		log.Printf("Received shutdown signal: %s. Stopping gRPC server...\n", sig.String())
		grpcServer.GracefulStop()
		log.Println("Shutdown complete.")
	}
}
