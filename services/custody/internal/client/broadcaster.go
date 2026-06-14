// Package client provides gRPC client wrappers for external services.
package client

import (
	"context"
	"fmt"
	"os"

	"github.com/p2p/custody/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/broadcaster"
	"github.com/p2p/shared/pb/blockchain/network"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/protobuf/types/known/anypb"
)

func getUSDCContractAddress() string {
	if envAddr := os.Getenv("USDC_CONTRACT_ADDRESS"); envAddr != "" {
		return envAddr
	}
	return "0x5FbDB2315678afecb367f032d93F642f64180aa3"
}

// BroadcasterClient wraps the gRPC BroadcastServiceClient and satisfies
// domain.Broadcaster.
type BroadcasterClient struct {
	conn   *grpc.ClientConn
	client broadcaster.BroadcastServiceClient
}

// Compile-time check that BroadcasterClient satisfies domain.Broadcaster.
var _ domain.Broadcaster = (*BroadcasterClient)(nil)

// NewBroadcasterClient dials the broadcaster gRPC service at the given address
// and returns a ready-to-use client.
func NewBroadcasterClient(addr string) (*BroadcasterClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("broadcaster grpc dial: %w", err)
	}
	return &BroadcasterClient{
		conn:   conn,
		client: broadcaster.NewBroadcastServiceClient(conn),
	}, nil
}

// SendTransfer builds an ERC-20 transfer intent and sends it to the
// broadcaster service via gRPC.
func (b *BroadcasterClient) SendTransfer(ctx context.Context, intent domain.TransferIntent) (*domain.TransferResult, error) {
	evmIntent := &broadcaster.EVMTransactionIntent{
		IntentType: broadcaster.EVMTransactionIntent_INTENT_TYPE_ERC20_TRANSFER,
		Intent: &broadcaster.EVMTransactionIntent_Erc20TransferIntent{
			Erc20TransferIntent: &broadcaster.ERC20TransferIntent{
				From:            intent.From,
				To:              intent.To,
				ContractAddress: getUSDCContractAddress(),
				Amount:          intent.Amount,
			},
		},
	}

	payload, err := anypb.New(evmIntent)
	if err != nil {
		return nil, fmt.Errorf("marshal intent payload: %w", err)
	}

	resp, err := b.client.SendTransaction(ctx, &broadcaster.TransactionIntentRequest{
		NetworkId:     network.NetworkId_NETWORK_ID_ETHEREUM,
		IntentType:    broadcaster.TransactionIntentRequest_INTENT_TYPE_TRANSFER,
		IntentPayload: payload,
	})
	if err != nil {
		return nil, fmt.Errorf("broadcaster SendTransaction: %w", err)
	}

	status := "pending"
	switch resp.GetStatus() {
	case broadcaster.TransactionIntentResponse_STATUS_BROADCASTED:
		status = "broadcasted"
	case broadcaster.TransactionIntentResponse_STATUS_BROADCAST_FAILED:
		return nil, fmt.Errorf("broadcast failed: %s", resp.GetErrorMessage())
	case broadcaster.TransactionIntentResponse_STATUS_SIGNING_FAILED:
		return nil, fmt.Errorf("signing failed: %s", resp.GetErrorMessage())
	case broadcaster.TransactionIntentResponse_STATUS_INVALID_REQUEST:
		return nil, fmt.Errorf("invalid request: %s", resp.GetErrorMessage())
	}

	return &domain.TransferResult{
		TxHash: resp.GetTransactionId(),
		Status: status,
	}, nil
}

// GetTokenBalance queries the on-chain ERC-20 token balance for the given
// address via the broadcaster → protocol → node chain.
func (b *BroadcasterClient) GetTokenBalance(ctx context.Context, address, contractAddress string) (string, error) {
	resp, err := b.client.GetTokenBalance(ctx, &broadcaster.GetTokenBalanceRequest{
		NetworkId:       network.NetworkId_NETWORK_ID_ETHEREUM,
		Address:         address,
		ContractAddress: contractAddress,
	})
	if err != nil {
		return "", fmt.Errorf("broadcaster GetTokenBalance: %w", err)
	}
	return resp.GetBalance(), nil
}

// Close tears down the underlying gRPC connection.
func (b *BroadcasterClient) Close() error {
	return b.conn.Close()
}
