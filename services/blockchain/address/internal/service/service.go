// Package service defines the protocol service for EVM chains
package service

import (
	"context"

	"github.com/p2p/blockchain/address/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/address"
)

// Config defines the configuration for the AddressService.
type Config struct {
	// Add configuration fields here if needed
}

// AddressService implements the address.AddressServiceServer interface.
type AddressService struct {
	address.UnimplementedAddressServiceServer
	repo domain.Repository
}

// NewAddressService creates a new AddressService instance.
func NewAddressService(repo domain.Repository) *AddressService {
	return &AddressService{
		repo: repo,
	}
}

func (s *AddressService) GetCustodyAddresses(ctx context.Context, req *address.GetCustodyAddressesRequest) (*address.GetCustodyAddressesResponse, error) {
	if req == nil {
		return &address.GetCustodyAddressesResponse{}, nil
	}

	custodyAddresses := s.repo.GetCustodyAddresses(ctx, req.GetAddresses())
	return &address.GetCustodyAddressesResponse{
		Addresses: custodyAddresses,
	}, nil
}
