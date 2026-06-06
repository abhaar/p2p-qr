package service

import (
	"context"
	"testing"
	"time"

	"github.com/p2p/blockchain/evm/protocol/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/address"
	network "github.com/p2p/shared/pb/blockchain/network"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

type mockAddressClient struct {
	address.AddressServiceClient
}

func (m *mockAddressClient) GetCustodyAddresses(ctx context.Context, in *address.GetCustodyAddressesRequest, opts ...grpc.CallOption) (*address.GetCustodyAddressesResponse, error) {
	return &address.GetCustodyAddressesResponse{
		Addresses: in.GetAddresses(),
	}, nil
}

func TestBlockchainService(t *testing.T) {
	// Initialize EVM client with the provided Alchemy Sepolia endpoint
	endpoint := "https://eth-sepolia.g.alchemy.com/v2/jkuYmSa1fVcwA5V0OlvFtk1nwIV_WaqE"
	client, err := domain.NewEVMClient(endpoint)
	if err != nil {
		t.Fatalf("failed to create EVM client: %v", err)
	}

	cfg := Config{
		GRPCServicePort:        ":50065",
		NetworkID:              network.NetworkId_NETWORK_ID_ETHEREUM,
		BlockchainNodeEndpoint: endpoint,
	}

	mockAddrClient := &mockAddressClient{}
	logger := zap.NewNop()
	svc := NewBlockchainService(client, mockAddrClient, cfg, logger)

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Test GetLatestBlock
	latestResp, err := svc.GetLatestBlock(ctx, &protocol.GetLatestBlockRequest{})
	if err != nil {
		t.Fatalf("GetLatestBlock failed: %v", err)
	}

	if latestResp.BlockHash == "" {
		t.Error("expected non-empty block hash")
	}
	if latestResp.BlockHeight == 0 {
		t.Error("expected block height > 0")
	}

	t.Logf("Latest block height: %d, hash: %s", latestResp.BlockHeight, latestResp.BlockHash)

	// Test GetBlockEvents
	// Try a few blocks starting from the latest block to get receipts/events
	var eventsResp *protocol.BlockchainEvents
	var eventsErr error
	targetHeight := latestResp.BlockHeight

	// Try up to 5 blocks going backward
	for i := range 5 {
		height := targetHeight - uint64(i)
		t.Logf("Trying GetBlockEvents for block height %d...", height)
		eventsResp, eventsErr = svc.GetBlockEvents(ctx, &protocol.GetBlockEventsRequest{
			BlockHeight: height,
		})
		if eventsErr == nil {
			break
		}
		t.Logf("Block height %d failed: %v", height, eventsErr)
	}

	if eventsErr != nil {
		t.Fatalf("GetBlockEvents failed for all tried blocks: %v", eventsErr)
	}

	if eventsResp.BlockHash == "" {
		t.Error("expected non-empty block hash in events response")
	}
	if eventsResp.NetworkId != network.NetworkId_NETWORK_ID_ETHEREUM {
		t.Errorf("expected network ID %v, got %v", network.NetworkId_NETWORK_ID_ETHEREUM, eventsResp.NetworkId)
	}

	t.Logf("Successfully retrieved events for block %d, found %d mapped events", eventsResp.BlockHeight, len(eventsResp.Events))
	for idx, event := range eventsResp.Events {
		t.Logf("  Event %d: TxHash: %s, Token: %s, From: %s, To: %s, Amount: %s",
			idx, event.TxHash, event.TokenId, event.From, event.To, event.Amount)
	}
}
