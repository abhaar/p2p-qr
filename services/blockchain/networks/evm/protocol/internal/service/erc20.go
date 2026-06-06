package service

import (
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/p2p/blockchain/evm/protocol/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/protocol"
)

var (
	logTransferSig     = []byte("Transfer(address,address,uint256)")
	logTransferSigHash = crypto.Keccak256Hash(logTransferSig)
)

type EventMapper func(log *types.Log, logIndex int) (*protocol.BlockchainEvent, error)

type LogTransfer struct {
	From  common.Address
	To    common.Address
	Value *big.Int
}

func getEventMapper(log *types.Log) (EventMapper, bool) {
	if log.Topics != nil && log.Topics[0].Hex() == logTransferSigHash.Hex() {
		return transferMapper, true
	}
	return nil, false
}

func transferMapper(log *types.Log, logIndex int) (*protocol.BlockchainEvent, error) {
	contractAbi, err := abi.JSON(strings.NewReader(string(domain.ContractABI)))
	if err != nil {
		return nil, err
	}

	var transferEvent LogTransfer
	err = contractAbi.UnpackIntoInterface(&transferEvent, "Transfer", log.Data)
	if err != nil {
		return nil, err
	}

	idempotencyID := fmt.Sprintf("%s:LogIndex:%d", log.TxHash.Hex(), logIndex)
	txHash := log.TxHash.Hex()
	tokenAddress := log.Address.Hex()
	transferEvent.From = common.HexToAddress(log.Topics[1].Hex())
	transferEvent.To = common.HexToAddress(log.Topics[2].Hex())

	return &protocol.BlockchainEvent{
		IdempotencyId: idempotencyID,
		TxHash:        txHash,
		TokenId:       tokenAddress,
		From:          transferEvent.From.Hex(),
		To:            transferEvent.To.Hex(),
		Amount:        transferEvent.Value.String(),
	}, nil
}
