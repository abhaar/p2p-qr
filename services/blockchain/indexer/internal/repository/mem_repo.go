// Package repository
package repository

import (
	"sync"

	"github.com/p2p/blockchain/indexer/v2/internal/domain"
)

type InMemoryRepo struct {
	mu               sync.RWMutex
	lastIndexedBlock domain.BlockHeight
}

func NewInMemoryRepo(startingBlock domain.BlockHeight) *InMemoryRepo {
	return &InMemoryRepo{
		lastIndexedBlock: startingBlock,
	}
}

func (r *InMemoryRepo) GetLastIndexedBlock() (domain.BlockHeight, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.lastIndexedBlock, nil
}

func (r *InMemoryRepo) SetLastIndexedBlock(block domain.BlockHeight) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.lastIndexedBlock = block
	return nil
}
