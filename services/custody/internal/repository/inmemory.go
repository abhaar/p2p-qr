// Package repository
package repository

import (
	"fmt"
	"math/big"
	"sync"

	"github.com/p2p/custody/v2/internal/domain"
	"go.uber.org/zap"
)

type InMemoryRepository struct {
	mu      sync.RWMutex
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
	if currency != "USDC" {
		return fmt.Errorf("only USDC is supported at this time. received: %s", currency)
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	oldBalance := new(big.Int)
	if bal, ok := r.balance[address]; ok {
		oldBalance = bal
	}

	newBalance := new(big.Int).Add(oldBalance, amount)

	r.logger.Info("updating ledger balance",
		zap.String("address", address),
		zap.Stringer("old_balance", oldBalance),
		zap.Stringer("new_balance", newBalance),
	)

	r.balance[address] = newBalance

	return nil
}

func (r *InMemoryRepository) GetBalance(address, currency string) (*big.Int, error) {
	if currency != "USDC" {
		return nil, fmt.Errorf("only USDC is supported at this time. received: %s", currency)
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	bal, ok := r.balance[address]
	if !ok {
		r.logger.Info("address not in ledger. returning 0", zap.String("address", address))
		return big.NewInt(0), nil
	}

	r.logger.Info("found balance for address in ledger",
		zap.String("address", address),
		zap.Stringer("balance", bal),
	)

	return bal, nil
}
