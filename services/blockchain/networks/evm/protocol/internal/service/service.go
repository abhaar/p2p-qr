// Package service defines the protocol service for EVM chains
package service

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/p2p/blockchain/evm/protocol/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/address"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"go.uber.org/zap"
)

// BlockchainService implements the protocol.ProtocolServiceServer interface for EVM networks.
type BlockchainService struct {
	protocol.UnimplementedProtocolServiceServer
	logger               *zap.Logger
	cfg                  Config
	evmClient            domain.EVMClient
	addressServiceClient address.AddressServiceClient
}

// NewBlockchainService creates a new BlockchainService instance.
func NewBlockchainService(client domain.EVMClient, addressServiceClient address.AddressServiceClient, cfg Config, logger *zap.Logger) *BlockchainService {
	return &BlockchainService{
		logger:               logger,
		cfg:                  cfg,
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
			NetworkId:   s.cfg.NetworkID,
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
		NetworkId:   s.cfg.NetworkID,
		Events:      relevantEvents,
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
