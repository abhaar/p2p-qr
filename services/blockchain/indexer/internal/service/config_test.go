package service

import (
	"flag"
	"os"
	"testing"

	network "github.com/p2p/shared/pb/blockchain/network"
)

func TestNewConfig(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	// Reset package-level flags to allow defining new ones during NewConfig call
	flag.CommandLine = flag.NewFlagSet(os.Args[0], flag.ExitOnError)

	os.Args = []string{
		"cmd",
		"-network-id", "ETHEREUM",
		"-protocol-service-endpoint", "http://localhost:8545",
		"-starting-block", "12345",
	}

	cfg, err := NewConfig()
	if err != nil {
		t.Fatalf("failed to create config: %v", err)
	}

	if cfg.NetworkID != network.NetworkId_NETWORK_ID_ETHEREUM {
		t.Errorf("expected NetworkID %v, got %v", network.NetworkId_NETWORK_ID_ETHEREUM, cfg.NetworkID)
	}
	if cfg.ProtocolServiceEndpoint != "http://localhost:8545" {
		t.Errorf("expected ProtocolServiceEndpoint 'http://localhost:8545', got '%s'", cfg.ProtocolServiceEndpoint)
	}
	if cfg.StartingBlock != 12345 {
		t.Errorf("expected StartingBlock 12345, got %d", cfg.StartingBlock)
	}
}
