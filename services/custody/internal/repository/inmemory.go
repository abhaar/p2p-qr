// Package repository
package repository

import (
	"fmt"
	"math/big"

	"github.com/p2p/custody/v2/internal/domain"
	"go.uber.org/zap"
)

type InMemoryRepository struct {
	logger  *zap.Logger
	balance map[string]*big.Int
}

func NewInMemoryRepository(logger *zap.Logger) domain.Repository {
	return &InMemoryRepository{
		logger:  logger,
		balance: make(map[string]*big.Int),
	}
}

func (r *InMemoryRepository) UpdateBalance(amount *big.Int, address, currency string) error {
	return nil
}

func (r *InMemoryRepository) GetBalance(address, currency string) (*big.Int, error) {
	if currency != "USDC" {
		return nil, fmt.Errorf("only USDC is supported at this time. received: %s", currency)
	}

	bal, ok := r.balance[address]
	if !ok {
		return nil, fmt.Errorf("address not custodied: %s", address)
	}

	return bal, nil
}
