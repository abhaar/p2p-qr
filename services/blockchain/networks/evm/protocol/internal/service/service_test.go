package service

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/p2p/blockchain/evm/protocol/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/address"
	"github.com/p2p/shared/pb/blockchain/broadcaster"
	network "github.com/p2p/shared/pb/blockchain/network"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"github.com/p2p/shared/pb/blockchain/signer"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/protobuf/types/known/anypb"
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
	svc := NewBlockchainService(client, mockAddrClient, cfg.NetworkID, logger)

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
	/*
	for idx, event := range eventsResp.Events {
		t.Logf("  Event %d: TxHash: %s, Token: %s, From: %s, To: %s, Amount: %s",
			idx, event.TxHash, event.TokenId, event.From, event.To, event.Amount)
	}
	*/

	// Test PrepareTransaction
	t.Run("PrepareTransaction Success", func(t *testing.T) {
		intent := &broadcaster.EVMTransactionIntent{
			IntentType: broadcaster.EVMTransactionIntent_INTENT_TYPE_ERC20_TRANSFER,
			Intent: &broadcaster.EVMTransactionIntent_Erc20TransferIntent{
				Erc20TransferIntent: &broadcaster.ERC20TransferIntent{
					From:            "0x93310a56147b1eA7486Ab84F8D850FD0A216429B",
					To:              "0x42F052F625E28c802f04978909Ece3f9d6e5E3a1",
					ContractAddress: "0x7b79995e5f793A07Bc00c21412e50Ecae098E7f9",
					Amount:          big.NewInt(1).Bytes(),
				},
			},
		}

		anyIntent, err := anypb.New(intent)
		if err != nil {
			t.Fatalf("failed to create anypb: %v", err)
		}

		resp, err := svc.PrepareTransaction(ctx, anyIntent)
		if err != nil {
			t.Fatalf("expected success, got error: %v", err)
		}
		if resp == nil {
			t.Error("expected non-nil response")
		}
	})

	t.Run("PrepareTransaction Insufficient Balance Failure", func(t *testing.T) {
		hugeAmount, _ := new(big.Int).SetString("10000000000000000000000000000000000000000000", 10)
		intent := &broadcaster.EVMTransactionIntent{
			IntentType: broadcaster.EVMTransactionIntent_INTENT_TYPE_ERC20_TRANSFER,
			Intent: &broadcaster.EVMTransactionIntent_Erc20TransferIntent{
				Erc20TransferIntent: &broadcaster.ERC20TransferIntent{
					From:            "0x93310a56147b1eA7486Ab84F8D850FD0A216429B",
					To:              "0x42F052F625E28c802f04978909Ece3f9d6e5E3a1",
					ContractAddress: "0x7b79995e5f793A07Bc00c21412e50Ecae098E7f9",
					Amount:          hugeAmount.Bytes(),
				},
			},
		}

		anyIntent, err := anypb.New(intent)
		if err != nil {
			t.Fatalf("failed to create anypb: %v", err)
		}

		_, err = svc.PrepareTransaction(ctx, anyIntent)
		if err == nil {
			t.Error("expected failure due to insufficient balance, got success")
		} else {
			t.Logf("Got expected failure: %v", err)
		}
	})

	t.Run("PrepareTransaction Invalid Address Failure", func(t *testing.T) {
		intent := &broadcaster.EVMTransactionIntent{
			IntentType: broadcaster.EVMTransactionIntent_INTENT_TYPE_ERC20_TRANSFER,
			Intent: &broadcaster.EVMTransactionIntent_Erc20TransferIntent{
				Erc20TransferIntent: &broadcaster.ERC20TransferIntent{
					From:            "invalid_address",
					To:              "0x42F052F625E28c802f04978909Ece3f9d6e5E3a1",
					ContractAddress: "0x7b79995e5f793A07Bc00c21412e50Ecae098E7f9",
					Amount:          big.NewInt(1).Bytes(),
				},
			},
		}

		anyIntent, err := anypb.New(intent)
		if err != nil {
			t.Fatalf("failed to create anypb: %v", err)
		}

		_, err = svc.PrepareTransaction(ctx, anyIntent)
		if err == nil {
			t.Error("expected failure due to invalid from address, got success")
		} else {
			t.Logf("Got expected failure: %v", err)
		}
	})

	// Test Transfer
	t.Run("Transfer Nil Request", func(t *testing.T) {
		_, err := svc.Transfer(ctx, nil)
		if err == nil {
			t.Error("expected error for nil request, got nil")
		}
	})

	t.Run("Transfer Invalid Signed Tx Data", func(t *testing.T) {
		resp, err := svc.Transfer(ctx, &signer.SignedTransaction{
			Data: []byte("invalid_data"),
		})
		if err != nil {
			t.Fatalf("did not expect gRPC error, got: %v", err)
		}
		if resp.Status != protocol.TransferResponse_TRANSFER_RESPONSE_FAILED {
			t.Errorf("expected failed status, got %v", resp.Status)
		}
		if resp.Error == "" {
			t.Error("expected non-empty error message")
		}
	})

	t.Run("Transfer Valid Sign but invalid parameters (e.g. low nonce)", func(t *testing.T) {
		// Construct and sign a dummy tx with go-ethereum
		privateKey, err := crypto.HexToECDSA("ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80")
		if err != nil {
			t.Fatalf("failed to parse private key: %v", err)
		}

		toAddr := common.HexToAddress("0x70997970C51812dc3A010C7d01b50e0d17dc79C8")
		tx := types.NewTx(&types.DynamicFeeTx{
			ChainID:   big.NewInt(31337), // Local Hardhat chain ID
			Nonce:     0,
			GasTipCap: big.NewInt(1000000000),
			GasFeeCap: big.NewInt(2000000000),
			Gas:       21000,
			To:        &toAddr,
			Value:     big.NewInt(1000000000),
			Data:      nil,
		})

		signerObj := types.NewLondonSigner(big.NewInt(31337))
		signedTx, err := types.SignTx(tx, signerObj, privateKey)
		if err != nil {
			t.Fatalf("failed to sign tx: %v", err)
		}

		txBytes, err := signedTx.MarshalBinary()
		if err != nil {
			t.Fatalf("failed to marshal tx: %v", err)
		}

		resp, err := svc.Transfer(ctx, &signer.SignedTransaction{
			Data: txBytes,
		})
		if err != nil {
			t.Fatalf("did not expect gRPC error, got: %v", err)
		}
		// Since we are running against Sepolia node (which doesn't support chain ID 31337),
		// the node should return an error. So it should fail to broadcast but still return the parsed transaction hash!
		if resp.Status != protocol.TransferResponse_TRANSFER_RESPONSE_FAILED {
			t.Errorf("expected failure status, got %v", resp.Status)
		}
		if resp.TxHash != signedTx.Hash().Hex() {
			t.Errorf("expected transaction hash %s, got %s", signedTx.Hash().Hex(), resp.TxHash)
		}
		t.Logf("Got expected broadcast failure for dummy transaction. Hash: %s, Error: %s", resp.TxHash, resp.Error)
	})
}
