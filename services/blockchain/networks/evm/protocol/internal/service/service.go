// Package service defines the protocol service for EVM chains
package service

import (
	"context"
	"fmt"
	"math/big"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/p2p/blockchain/evm/protocol/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/address"
	"github.com/p2p/shared/pb/blockchain/broadcaster"
	network "github.com/p2p/shared/pb/blockchain/network"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"github.com/p2p/shared/pb/blockchain/signer"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/anypb"
)

// BlockchainService implements the protocol.ProtocolServiceServer interface for EVM networks.
type BlockchainService struct {
	protocol.UnimplementedProtocolServiceServer
	logger               *zap.Logger
	networkID            network.NetworkId
	evmClient            domain.EVMClient
	addressServiceClient address.AddressServiceClient
}

// NewBlockchainService creates a new BlockchainService instance.
func NewBlockchainService(client domain.EVMClient, addressServiceClient address.AddressServiceClient, networkId network.NetworkId, logger *zap.Logger) *BlockchainService {
	return &BlockchainService{
		logger:               logger,
		networkID:            networkId,
		evmClient:            client,
		addressServiceClient: addressServiceClient,
	}
}

// GetLatestBlock retrieves the latest block from the EVM network.
func (s *BlockchainService) GetLatestBlock(ctx context.Context, req *protocol.GetLatestBlockRequest) (*protocol.GetLatestBlockResponse, error) {
	var latestBlockNumber hexutil.Uint64
	err := s.evmClient.RPCClient().CallContext(ctx, &latestBlockNumber, "eth_blockNumber")
	if err != nil {
		return nil, err
	}

	header, err := s.getHeaderByNumber(ctx, latestBlockNumber.String())
	if err != nil {
		return nil, err
	}

	return &protocol.GetLatestBlockResponse{
		BlockHash:   header.Hash().String(),
		BlockHeight: header.Number.Uint64(),
	}, nil
}

// GetBlockEvents retrieves value-transfer events (transactions) in a block by height.
func (s *BlockchainService) GetBlockEvents(ctx context.Context, req *protocol.GetBlockEventsRequest) (*protocol.BlockchainEvents, error) {
	if req == nil {
		return nil, fmt.Errorf("GetBlockEventsRequest must not be nil")
	}

	var receipts []*types.Receipt
	blockHeightHexStr := hexutil.EncodeUint64(req.GetBlockHeight())
	err := s.evmClient.RPCClient().CallContext(ctx, &receipts, "eth_getBlockReceipts", blockHeightHexStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipts for block %d: %s", req.GetBlockHeight(), err.Error())
	}

	if len(receipts) == 0 {
		header, err := s.getHeaderByNumber(ctx, blockHeightHexStr)
		if err != nil {
			return nil, fmt.Errorf("failed to get block header for empty block %d: %w", req.GetBlockHeight(), err)
		}
		return &protocol.BlockchainEvents{
			BlockHash:   header.Hash().String(),
			BlockHeight: req.GetBlockHeight(),
			NetworkId:   s.networkID,
			Events:      nil,
		}, nil
	}
	blockHash := receipts[0].BlockHash.String()
	blockNumber := receipts[0].BlockNumber.Uint64()

	// TODO: DO NOT USE Transport Layer Message. Replace With Domain Data Models.
	var events []*protocol.BlockchainEvent
	for _, receipt := range receipts {
		if receipt.Status != types.ReceiptStatusSuccessful {
			continue
		}

		for i, log := range receipt.Logs {
			if mapper, ok := getEventMapper(log); ok {
				blockchainEvent, err := mapper(log, i)
				if err != nil {
					return nil, err
				}
				events = append(events, blockchainEvent)
			}
		}
	}

	relevantEvents, err := s.filterRelevantEvents(ctx, events)
	if err != nil {
		s.logger.Error("failed to filter events", zap.Error(err))
		return nil, err
	}

	return &protocol.BlockchainEvents{
		BlockHash:   blockHash,
		BlockHeight: blockNumber,
		NetworkId:   s.networkID,
		Events:      relevantEvents,
	}, nil
}

func (s *BlockchainService) PrepareTransaction(ctx context.Context, in *anypb.Any) (*signer.UnsignedEvmTransaction, error) {
	if in == nil {
		return nil, fmt.Errorf("prepare transaction request must not be empty")
	}

	if !in.MessageIs((*broadcaster.ERC20TransferIntent)(nil)) {
		s.logger.Error("invalid transaction type received for prepare transaction", zap.String("type", in.GetTypeUrl()))
		return nil, fmt.Errorf("invalid transaction type received: %s", in.GetTypeUrl())
	}

	erc20TransferIntent := &broadcaster.ERC20TransferIntent{}
	err := anypb.UnmarshalTo(in, erc20TransferIntent, proto.UnmarshalOptions{})
	if err != nil {
		s.logger.Error("failed to unmarshal intent into ERC20TransferIntent", zap.Error(err))
		return nil, fmt.Errorf("invalid transaction type received: %s", in.GetTypeUrl())
	}

	s.logger.Info("request to validate erc-20 transfer received")
	if err := s.validateErc20TransferRequest(ctx, erc20TransferIntent); err != nil {
		return nil, err
	}

	if !common.IsHexAddress(erc20TransferIntent.GetFrom()) {
		return nil, fmt.Errorf("invalid from address: %s", erc20TransferIntent.GetFrom())
	}
	if !common.IsHexAddress(erc20TransferIntent.GetTo()) {
		return nil, fmt.Errorf("invalid to address: %s", erc20TransferIntent.GetTo())
	}
	if !common.IsHexAddress(erc20TransferIntent.GetContractAddress()) {
		return nil, fmt.Errorf("invalid contract address: %s", erc20TransferIntent.GetContractAddress())
	}
	if erc20TransferIntent.GetAmount() == "" {
		return nil, fmt.Errorf("amount must not be empty")
	}

	from := common.HexToAddress(erc20TransferIntent.GetFrom())
	to := common.HexToAddress(erc20TransferIntent.GetContractAddress())
	recipient := common.HexToAddress(erc20TransferIntent.GetTo())

	amount, ok := new(big.Int).SetString(erc20TransferIntent.GetAmount(), 10)
	if !ok {
		s.logger.Error("failed to parse amount as base 10", zap.String("amount", erc20TransferIntent.GetAmount()))
	}

	contractAbi, err := abi.JSON(strings.NewReader(domain.ContractABI))
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	data, err := contractAbi.Pack("transfer", recipient, amount)
	if err != nil {
		return nil, fmt.Errorf("failed to pack transfer input: %w", err)
	}

	nonce, err := s.getNonce(ctx, from)
	if err != nil {
		s.logger.Error("failed to get nonce", zap.Error(err))
		return nil, err
	}

	gasLimit, err := s.getGasLimit(ctx, from, to, data)
	if err != nil {
		s.logger.Error("failed to calculate gas limit", zap.Error(err))
		return nil, err
	}

	maxFeePerGas, maxPriorityFeePerGas, err := s.getFees(ctx)
	if err != nil {
		s.logger.Error("failed to calculate fees", zap.Error(err))
		return nil, err
	}

	return &signer.UnsignedEvmTransaction{
		NetworkId:            s.networkID,
		From:                 from.Hex(),
		Nonce:                nonce,
		To:                   to.Hex(),
		GasLimit:             gasLimit,
		MaxPriorityFeePerGas: maxPriorityFeePerGas.Bytes(),
		MaxFeePerGas:         maxFeePerGas.Bytes(),
		Data:                 data,
	}, nil
}

func (s *BlockchainService) Transfer(ctx context.Context, req *signer.SignedTransaction) (*protocol.TransferResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	var tx types.Transaction
	if err := tx.UnmarshalBinary(req.GetData()); err != nil {
		s.logger.Error("failed to unmarshal signed transaction data", zap.Error(err))
		return &protocol.TransferResponse{
			Status: protocol.TransferResponse_TRANSFER_RESPONSE_FAILED,
			Error:  fmt.Sprintf("invalid signed transaction: %s", err.Error()),
		}, nil
	}

	txHash := tx.Hash().Hex()

	var txHashStr string
	err := s.evmClient.RPCClient().CallContext(ctx, &txHashStr, "eth_sendRawTransaction", hexutil.Encode(req.GetData()))
	if err != nil {
		s.logger.Error("failed to broadcast transaction", zap.String("tx_hash", txHash), zap.Error(err))
		return &protocol.TransferResponse{
			Status: protocol.TransferResponse_TRANSFER_RESPONSE_FAILED,
			TxHash: txHash,
			Error:  err.Error(),
		}, nil
	}

	s.logger.Info("transaction broadcasted successfully", zap.String("tx_hash", txHashStr))
	return &protocol.TransferResponse{
		Status: protocol.TransferResponse_TRANSFER_RESPONSE_BROADCASTED,
		TxHash: txHashStr,
	}, nil
}

func (s *BlockchainService) getHeaderByNumber(ctx context.Context, number string) (*types.Header, error) {
	var header *types.Header
	err := s.evmClient.RPCClient().CallContext(ctx, &header, "eth_getBlockByNumber", number, false)
	if err != nil {
		return nil, err
	}

	if header == nil {
		return nil, fmt.Errorf("block header not found for block number: %s", number)
	}

	return header, nil
}

func (s *BlockchainService) filterRelevantEvents(ctx context.Context, events []*protocol.BlockchainEvent) ([]*protocol.BlockchainEvent, error) {
	addressSet := make(map[string]struct{})
	for _, event := range events {
		addressSet[event.GetFrom()] = struct{}{}
		addressSet[event.GetTo()] = struct{}{}
	}

	addressList := make([]string, 0)
	for address := range addressSet {
		addressList = append(addressList, address)
	}

	relevantAddresses, err := s.addressServiceClient.GetCustodyAddresses(ctx, &address.GetCustodyAddressesRequest{
		Addresses: addressList,
	})
	if err != nil {
		return nil, err
	}

	relevantAddressSet := make(map[string]struct{})
	for _, address := range relevantAddresses.GetAddresses() {
		relevantAddressSet[address] = struct{}{}
	}

	filteredEvents := make([]*protocol.BlockchainEvent, 0)
	for _, event := range events {
		if _, ok := relevantAddressSet[event.GetFrom()]; ok {
			filteredEvents = append(filteredEvents, event)
			continue
		}

		if _, ok := relevantAddressSet[event.GetTo()]; ok {
			filteredEvents = append(filteredEvents, event)
		}
	}

	s.logger.Info("filtered out irrelevant blockchain events",
		zap.Int("total", len(events)),
		zap.Int("relevant", len(filteredEvents)),
	)

	return filteredEvents, nil
}

func (s *BlockchainService) validateErc20TransferRequest(ctx context.Context, req *broadcaster.ERC20TransferIntent) error {
	if req == nil {
		return fmt.Errorf("request must not be nil")
	}

	if !common.IsHexAddress(req.GetFrom()) {
		return fmt.Errorf("invalid from address: %s", req.GetFrom())
	}
	if !common.IsHexAddress(req.GetTo()) {
		return fmt.Errorf("invalid to address: %s", req.GetTo())
	}
	if !common.IsHexAddress(req.GetContractAddress()) {
		return fmt.Errorf("invalid contract address: %s", req.GetContractAddress())
	}
	if len(req.GetAmount()) == 0 {
		return fmt.Errorf("amount must not be empty")
	}

	fromAddr := common.HexToAddress(req.GetFrom())
	toAddr := common.HexToAddress(req.GetTo())
	contractAddr := common.HexToAddress(req.GetContractAddress())

	amount, ok := new(big.Int).SetString(req.GetAmount(), 10)
	if !ok {
		s.logger.Error("failed to parse amount as base 10", zap.String("amount", req.GetAmount()))
	}

	contractAbi, err := abi.JSON(strings.NewReader(domain.ContractABI))
	if err != nil {
		return fmt.Errorf("failed to parse contract ABI: %w", err)
	}

	data, err := contractAbi.Pack("transfer", toAddr, amount)
	if err != nil {
		return fmt.Errorf("failed to pack transfer input: %w", err)
	}

	callArg := map[string]interface{}{
		"from": fromAddr.Hex(),
		"to":   contractAddr.Hex(),
		"data": hexutil.Bytes(data),
	}

	var result hexutil.Bytes
	err = s.evmClient.RPCClient().CallContext(ctx, &result, "eth_call", callArg, "latest")
	if err != nil {
		return fmt.Errorf("simulation failed: %w", err)
	}

	if len(result) > 0 {
		var transferSuccess bool
		err = contractAbi.UnpackIntoInterface(&transferSuccess, "transfer", result)
		if err == nil && !transferSuccess {
			return fmt.Errorf("simulation succeeded but returned false")
		}
	}

	return nil
}

func (s *BlockchainService) getNonce(ctx context.Context, account common.Address) (uint64, error) {
	var nonce hexutil.Uint64
	err := s.evmClient.RPCClient().CallContext(ctx, &nonce, "eth_getTransactionCount", account.Hex(), "latest")
	if err != nil {
		return 0, err
	}
	return uint64(nonce), nil
}

func (s *BlockchainService) getGasLimit(ctx context.Context, from, to common.Address, data []byte) (uint64, error) {
	callArg := map[string]interface{}{
		"from": from.Hex(),
		"to":   to.Hex(),
		"data": hexutil.Bytes(data),
	}
	var gas hexutil.Uint64
	err := s.evmClient.RPCClient().CallContext(ctx, &gas, "eth_estimateGas", callArg)
	if err != nil {
		return 0, err
	}
	return uint64(gas), nil
}

func (s *BlockchainService) getFees(ctx context.Context) (*big.Int, *big.Int, error) {
	var maxPriorityFee hexutil.Big
	err := s.evmClient.RPCClient().CallContext(ctx, &maxPriorityFee, "eth_maxPriorityFeePerGas")
	if err != nil {
		return nil, nil, err
	}

	header, err := s.getHeaderByNumber(ctx, "latest")
	if err != nil {
		return nil, nil, err
	}

	baseFee := header.BaseFee
	if baseFee == nil {
		// Fallback for non-EIP1559 networks: use eth_gasPrice
		var gasPrice hexutil.Big
		err = s.evmClient.RPCClient().CallContext(ctx, &gasPrice, "eth_gasPrice")
		if err != nil {
			return nil, nil, err
		}
		return (*big.Int)(&gasPrice), (*big.Int)(&maxPriorityFee), nil
	}

	// MaxFeePerGas = 2 * BaseFee + MaxPriorityFeePerGas
	maxFeePerGas := new(big.Int).Mul(baseFee, big.NewInt(2))
	maxFeePerGas.Add(maxFeePerGas, (*big.Int)(&maxPriorityFee))

	return maxFeePerGas, (*big.Int)(&maxPriorityFee), nil
}
