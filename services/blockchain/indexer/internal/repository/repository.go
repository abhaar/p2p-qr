package repository

import "github.com/p2p/blockchain/indexer/v2/internal/domain"

type Repository interface {
	GetLastIndexedBlock() (domain.BlockHeight, error)
	SetLastIndexedBlock(domain.BlockHeight) error
}
