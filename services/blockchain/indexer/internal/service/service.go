// Package service contains indexer service
package service

import (
	"context"
	"time"

	"github.com/p2p/blockchain/indexer/v2/internal/domain"
	"github.com/p2p/blockchain/indexer/v2/internal/publisher"
	"github.com/p2p/blockchain/indexer/v2/internal/repository"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"go.uber.org/zap"
)

type Indexer struct {
	logger         *zap.Logger
	protocolClient protocol.ProtocolServiceClient
	repo           repository.Repository
	publisher      publisher.EventPublisher
}

func New(logger *zap.Logger, client protocol.ProtocolServiceClient, repo repository.Repository, publisher publisher.EventPublisher) *Indexer {
	return &Indexer{
		logger:         logger,
		protocolClient: client,
		repo:           repo,
		publisher:      publisher,
	}
}

func (s *Indexer) Run(ctx context.Context) error {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			lasIndexedBlock, err := s.getLastIndexedBlock(ctx)
			if err != nil {
				return err
			}

			tip, err := s.getChainTip(ctx)
			if err != nil {
				return err
			}

			for i := lasIndexedBlock; i <= tip; i++ {
				blockHeightUint64 := uint64(i)
				events, err := s.protocolClient.GetBlockEvents(ctx, &protocol.GetBlockEventsRequest{BlockHeight: blockHeightUint64})
				if err != nil {
					s.logger.Error("failed to get events for block", zap.Uint64("height", blockHeightUint64), zap.Error(err))
					return err
				}

				err = s.publisher.Publish(ctx, events)
				if err != nil {
					s.logger.Error("failed to publish events for block", zap.Uint64("height", blockHeightUint64), zap.Error(err))
					return err
				}

				err = s.repo.SetLastIndexedBlock(i)
				if err != nil {
					s.logger.Error("failed to update scanning progress", zap.Uint64("block", blockHeightUint64))
				}
			}
		}
	}
}

func (s *Indexer) getLastIndexedBlock(ctx context.Context) (domain.BlockHeight, error) {
	lasIndexedBlock, _ := s.repo.GetLastIndexedBlock()
	if lasIndexedBlock == 0 {
		tip, err := s.getChainTip(ctx)
		if err != nil {
			s.logger.Error("failed to get chain tip", zap.Error(err))
			return domain.BlockHeight(0), err
		}

		_ = s.repo.SetLastIndexedBlock(lasIndexedBlock)

		return tip, nil
	}

	return lasIndexedBlock, nil
}

func (s *Indexer) getChainTip(ctx context.Context) (domain.BlockHeight, error) {
	tip, err := s.protocolClient.GetLatestBlock(ctx, &protocol.GetLatestBlockRequest{})
	if err != nil {
		s.logger.Error("failed to get chain tip", zap.Error(err))
		return domain.BlockHeight(0), err
	}

	return domain.BlockHeight(tip.GetBlockHeight()), nil
}
