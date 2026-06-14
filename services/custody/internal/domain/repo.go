package domain

import (
	"math/big"
)

type Repository interface {
	UpdateBalance(amount *big.Int, address, currency string) error
	GetBalance(address, currency string) (*big.Int, error)
}
