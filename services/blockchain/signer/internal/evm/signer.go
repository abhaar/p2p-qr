// Package evm
package evm

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/p2p/blockchain/signer/v2/api"
)

const (
	chainID = 31337
	pk      = "ac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
)

func SignErc20TransferIntent(intent api.TransactionIntent) (*api.SignTransactionResponse, error) {
	if intent.Transfer == nil {
		return nil, fmt.Errorf("transfer intent details are missing")
	}

	erc20TransferIntent := intent.Transfer

	toAddr := common.HexToAddress(erc20TransferIntent.To)
	amount := erc20TransferIntent.Amount

	parsedABI, err := ContractMetaData.GetAbi()
	if err != nil {
		return nil, fmt.Errorf("failed to parse erc20 contract abi: %w", err)
	}

	data, err := parsedABI.Pack("transfer", toAddr, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to pack erc20 data: %w", err)
	}

	tokenAddr := common.HexToAddress(erc20TransferIntent.ContractAddress)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(chainID),
		Nonce:     erc20TransferIntent.Nonce,
		GasTipCap: erc20TransferIntent.MaxPriorityFeePerGas,
		GasFeeCap: erc20TransferIntent.MaxFeePerGas,
		Gas:       erc20TransferIntent.GasLimit,
		To:        &tokenAddr,
		Data:      data,
	})

	privateKey, err := crypto.HexToECDSA(pk)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Sign Transaction
	signer := types.NewLondonSigner(big.NewInt(chainID))
	signedTx, err := types.SignTx(tx, signer, privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign transaction: %w", err)
	}

	signedTxBytes, err := signedTx.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to marshal signed transaction: %w", err)
	}

	return &api.SignTransactionResponse{
		SignedTx: signedTxBytes,
	}, nil
}
