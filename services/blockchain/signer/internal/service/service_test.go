package service

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/p2p/blockchain/signer/v2/api"
	"github.com/p2p/shared/pb/blockchain/network"
)

func TestSignTransaction_ERC20Transfer(t *testing.T) {
	signer := &InMemorySigner{}

	req := &api.SignTransactionRequest{
		NetworkID: network.NetworkId_NETWORK_ID_ETHEREUM,
		Intent: api.TransactionIntent{
			Type: api.IntentTypeErc20Transfer,
			Transfer: &api.ERC20TransferIntent{
				ContractAddress:      "0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48", // USDC
				To:                   "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
				Amount:               big.NewInt(500000000),
				Nonce:                10,
				GasLimit:             65000,
				MaxFeePerGas:          big.NewInt(45000000000),
				MaxPriorityFeePerGas: big.NewInt(2000000000),
			},
		},
	}

	resp, err := signer.SignTransaction(context.Background(), req)
	if err != nil {
		t.Fatalf("failed to sign transaction: %v", err)
	}

	if len(resp.SignedTx) == 0 {
		t.Fatal("expected signed transaction bytes, got empty slice")
	}

	// Verify the signed transaction
	var tx types.Transaction
	err = tx.UnmarshalBinary(resp.SignedTx)
	if err != nil {
		t.Fatalf("failed to unmarshal signed transaction: %v", err)
	}

	if tx.Type() != types.DynamicFeeTxType {
		t.Errorf("expected transaction type %d (DynamicFeeTx), got %d", types.DynamicFeeTxType, tx.Type())
	}

	// Verify the To field is the token contract address
	expectedTo := common.HexToAddress("0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48")
	if *tx.To() != expectedTo {
		t.Errorf("expected to token address %s, got %s", expectedTo.Hex(), tx.To().Hex())
	}

	// Value sent should be 0
	if tx.Value().Sign() != 0 {
		t.Errorf("expected value to be 0 for token transfer, got %s", tx.Value().String())
	}

	// Validate the data payload contains the expected transfer signature
	data := tx.Data()
	if len(data) != 68 { // 4 bytes method ID + 32 bytes address + 32 bytes amount
		t.Fatalf("expected ERC20 transfer data length 68, got %d", len(data))
	}

	methodID := data[:4]
	expectedMethodID := []byte{0xa9, 0x05, 0x9c, 0xbb} // transfer(address,uint256)
	for i := range methodID {
		if methodID[i] != expectedMethodID[i] {
			t.Errorf("unexpected method ID bytes at %d", i)
		}
	}

	// Extract the sender address
	chainID := big.NewInt(31337)
	ethSigner := types.NewLondonSigner(chainID)
	sender, err := types.Sender(ethSigner, &tx)
	if err != nil {
		t.Fatalf("failed to recover sender: %v", err)
	}

	expectedSender := common.HexToAddress("0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266")
	if sender != expectedSender {
		t.Errorf("expected sender %s, got %s", expectedSender.Hex(), sender.Hex())
	}
}

func TestSignTransaction_NilRequest(t *testing.T) {
	signer := &InMemorySigner{}
	_, err := signer.SignTransaction(context.Background(), nil)
	if err == nil {
		t.Error("expected error when request is nil, got nil")
	}
}
