// Package service implements the business logic for the custody service.
package service

import (
	"context"

	"go.uber.org/zap"

	"github.com/p2p/custody/v2/internal/domain"
)

// CustodyService is the concrete implementation of domain.CustodyService.
type CustodyService struct {
	logger      *zap.Logger
	broadcaster domain.Broadcaster
}

// NewCustodyService creates a new CustodyService.
func NewCustodyService(logger *zap.Logger, broadcaster domain.Broadcaster) *CustodyService {
	return &CustodyService{
		logger:      logger,
		broadcaster: broadcaster,
	}
}

// Compile-time check that CustodyService satisfies the domain interface.
var _ domain.CustodyService = (*CustodyService)(nil)

// GetBalance returns the USDC balance for the given address.
func (s *CustodyService) GetBalance(ctx context.Context, address, currency string) (*domain.Balance, error) {
	s.logger.Info("fetching balance",
		zap.String("address", address),
		zap.String("currency", currency),
	)

	// TODO: replace stub with on-chain USDC balance lookup.
	return &domain.Balance{
		Address:  address,
		Currency: currency,
		Amount:   "0",
	}, nil
}

// Transfer initiates a token transfer by delegating to the broadcaster service.
func (s *CustodyService) Transfer(ctx context.Context, intent domain.TransferIntent) (*domain.TransferResult, error) {
	s.logger.Info("initiating transfer",
		zap.String("from", intent.From),
		zap.String("to", intent.To),
		zap.String("amount", intent.Amount),
		zap.String("currency", intent.Currency),
	)

	result, err := s.broadcaster.SendTransfer(ctx, intent)
	if err != nil {
		s.logger.Error("transfer failed", zap.Error(err))
		return nil, err
	}

	s.logger.Info("transfer submitted",
		zap.String("tx_hash", result.TxHash),
		zap.String("status", result.Status),
	)

	return result, nil
}
