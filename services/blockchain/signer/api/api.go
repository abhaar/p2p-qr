// Package api contains the definitions on how to interact with the signer service.
package api

import (
	"context"
	"math/big"

	"github.com/p2p/shared/pb/blockchain/network"
)

type IntentType string

const (
	IntentTypeErc20Transfer IntentType = "ERC20_TRANSFER"
	IntentTypeContractCall  IntentType = "CONTRACT_CALL"
)

// ERC20TransferIntent represents a standard transfer of native assets or tokens.
type ERC20TransferIntent struct {
	ContractAddress      string
	To                   string
	Amount               *big.Int
	Nonce                uint64
	GasLimit             uint64
	MaxFeePerGas         *big.Int
	MaxPriorityFeePerGas *big.Int
}

// ContractCallIntent represents a contract execution/interaction.
type ContractCallIntent struct {
	To    string `json:"to"`    // Contract address
	Data  []byte `json:"data"`  // Contract input bytes
	Value string `json:"value"` // Optional native asset value to attach
}

// TransactionIntent is a polymorphic container for transaction signing intents.
type TransactionIntent struct {
	Type IntentType `json:"type"`

	// Exactly one of the following should be populated based on the Type.
	Transfer     *ERC20TransferIntent `json:"transfer,omitempty"`
	ContractCall *ContractCallIntent  `json:"contract_call,omitempty"`
}

// SignTransactionRequest contains all parameters required to sign a transaction intent.
type SignTransactionRequest struct {
	NetworkID network.NetworkId `json:"network_id"`
	Intent    TransactionIntent `json:"intent"`
}

// SignTransactionResponse returns the resulting signed payload and optional metadata.
type SignTransactionResponse struct {
	SignedTx []byte `json:"signed_tx"`
}

// Signer defines the generic multi-chain transaction signing interface.
type Signer interface {
	SignTransaction(ctx context.Context, req *SignTransactionRequest) (*SignTransactionResponse, error)
}
