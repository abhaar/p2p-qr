// Package service defines the protocol service for EVM chains
package service

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/p2p/blockchain/evm/protocol/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/protocol"
)

// BlockchainService implements the protocol.ProtocolServiceServer interface for EVM networks.
type BlockchainService struct {
	protocol.UnimplementedProtocolServiceServer
	cfg    Config
	client domain.EVMClient
}

// NewBlockchainService creates a new BlockchainService instance.
func NewBlockchainService(client domain.EVMClient, cfg Config) *BlockchainService {
	return &BlockchainService{
		cfg:    cfg,
		client: client,
	}
}

// GetLatestBlock retrieves the latest block from the EVM network.
func (s *BlockchainService) GetLatestBlock(ctx context.Context, req *protocol.GetLatestBlockRequest) (*protocol.GetLatestBlockResponse, error) {
	var latestBlockNumber hexutil.Uint64
	err := s.client.RPCClient().CallContext(ctx, &latestBlockNumber, "eth_blockNumber")
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
	err := s.client.RPCClient().CallContext(ctx, &receipts, "eth_getBlockReceipts", blockHeightHexStr)
	if err != nil {
		return nil, fmt.Errorf("failed to get receipts for block %d: %s", req.GetBlockHeight(), err.Error())
	}

	if len(receipts) == 0 {
		return nil, fmt.Errorf("receipts not found for block: %d", req.GetBlockHeight())
	}
	blockHash := receipts[0].BlockHash.String()
	blockNumber := receipts[0].BlockNumber.Uint64()

	// filter relevant events
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

	return &protocol.BlockchainEvents{
		BlockHash:   blockHash,
		BlockHeight: blockNumber,
		NetworkId:   s.cfg.NetworkID,
		Events:      events,
	}, nil
}

func (s *BlockchainService) getHeaderByNumber(ctx context.Context, number string) (*types.Header, error) {
	var header *types.Header
	err := s.client.RPCClient().CallContext(ctx, &header, "eth_getBlockByNumber", number, false)
	if err != nil {
		return nil, err
	}

	if header == nil {
		return nil, fmt.Errorf("block header not found for block number: %s", number)
	}

	return header, nil
}
