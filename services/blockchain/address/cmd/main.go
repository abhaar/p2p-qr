package main

import (
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"

	"github.com/p2p/blockchain/address/v2/internal/repository"
	"github.com/p2p/blockchain/address/v2/internal/service"
	"github.com/p2p/shared/pb/blockchain/address"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const defaultPort = ":50064"

func main() {
	log.Println("Address service starting up...")

	// Listen on default port
	lis, err := net.Listen("tcp", defaultPort)
	if err != nil {
		log.Fatalf("failed to bind grpc server: %v", err)
	}

	addressServer := service.NewAddressService(repository.NewInMemoryRepository())

	grpcServer := grpc.NewServer()
	address.RegisterAddressServiceServer(grpcServer, addressServer)
	reflection.Register(grpcServer)

	errChan := make(chan error, 1)

	go func() {
		log.Printf("gRPC server started listening on port %s\n", defaultPort)
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
