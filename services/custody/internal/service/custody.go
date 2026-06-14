// Package service implements the business logic for the custody service.
package service

import (
	"context"
	"fmt"
	"math/big"
	"os"

	"go.uber.org/zap"

	"github.com/p2p/custody/v2/internal/domain"
	"github.com/p2p/custody/v2/internal/repository"
)

// defaultUsdcContractAddress is the USDC contract address deployed on local Anvil by default.
const defaultUsdcContractAddress = "0x5FbDB2315678afecb367f032d93F642f64180aa3"

// CustodyService is the concrete implementation of domain.CustodyService.
type CustodyService struct {
	logger         *zap.Logger
	broadcaster    domain.Broadcaster
	addressService domain.AddressService
	repo           domain.Repository
}

// NewCustodyService creates a new CustodyService.
func NewCustodyService(logger *zap.Logger, broadcaster domain.Broadcaster, addressService domain.AddressService) *CustodyService {
	repo := repository.NewInMemoryRepository(logger)
	service := &CustodyService{
		logger:         logger,
		broadcaster:    broadcaster,
		addressService: addressService,
		repo:           repo,
	}

	custodyAddresses, err := addressService.GetCustodyAddresses(context.Background(), nil)
	if err != nil {
		logger.Fatal("failed to fetch custody addresses from address service", zap.Error(err))
	}

	for _, addr := range custodyAddresses {
		balance, err := service.getBalance(context.Background(), addr, "USDC")
		if err != nil {
			logger.Fatal("failed to initialize wallet balance", zap.String("address", addr), zap.Error(err))
		}

		balanceBigInt, ok := new(big.Int).SetString(balance, 10)
		if !ok {
			logger.Fatal("failed to parse initial wallet balance", zap.String("address", addr), zap.String("balance", balance))
		}

		err = service.repo.UpdateBalance(balanceBigInt, addr, "USDC")
		if err != nil {
			logger.Fatal("failed to store initial balance", zap.String("address", addr), zap.String("balance", balance), zap.Error(err))
		}
	}

	return service
}

// Compile-time check that CustodyService satisfies the domain interface.
var _ domain.CustodyService = (*CustodyService)(nil)

func getUSDCContractAddress() string {
	if envAddr := os.Getenv("USDC_CONTRACT_ADDRESS"); envAddr != "" {
		return envAddr
	}
	return defaultUsdcContractAddress
}

// GetBalance returns the token balance for the given address by querying the
// broadcaster service, which in turn calls the protocol service to read the
// balance from the blockchain node.
func (s *CustodyService) GetBalance(ctx context.Context, address, currency string) (domain.Balance, error) {
	balance, err := s.repo.GetBalance(address, currency)
	if err != nil {
		s.logger.Error("failed to fetch balance", zap.Error(err))
		return domain.Balance{}, err
	}

	decimals := big.NewInt(1000000)
	normalizedAmount := new(big.Int)
	normalizedAmount.Div(balance, decimals)

	return domain.Balance{
		Address:  address,
		Currency: currency,
		Amount:   normalizedAmount.String(),
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

func (s *CustodyService) getBalance(ctx context.Context, address, currency string) (string, error) {
	s.logger.Info("fetching balance",
		zap.String("address", address),
		zap.String("currency", currency),
	)

	var contractAddress string
	if currency == "USDC" {
		contractAddress = getUSDCContractAddress()
	} else {
		return "", fmt.Errorf("unsupported currency: %s", currency)
	}

	amount, err := s.broadcaster.GetTokenBalance(ctx, address, contractAddress)
	if err != nil {
		s.logger.Error("failed to fetch balance", zap.Error(err))
		return "", err
	}

	return amount, nil
}

// Repo returns the underlying balance repository.
func (s *CustodyService) Repo() domain.Repository {
	return s.repo
}
