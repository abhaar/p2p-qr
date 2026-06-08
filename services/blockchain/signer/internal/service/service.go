// Package service defines the signer service.
package service

import (
	"context"
	"fmt"

	"github.com/p2p/blockchain/signer/v2/api"
	"github.com/p2p/blockchain/signer/v2/internal/evm"
	"github.com/p2p/shared/pb/blockchain/network"
)

var _ api.Signer = (*InMemorySigner)(nil)

type InMemorySigner struct{}

func (s *InMemorySigner) SignTransaction(ctx context.Context, req *api.SignTransactionRequest) (*api.SignTransactionResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	if req.NetworkID != network.NetworkId_NETWORK_ID_ETHEREUM {
		return nil, fmt.Errorf("unsupported network: %s", req.NetworkID.String())
	}

	switch req.Intent.Type {
	case api.IntentTypeErc20Transfer:
		return evm.SignErc20TransferIntent(req.Intent)

	default:
		return nil, fmt.Errorf("unsupported intent type on Ethereum: %s", req.Intent.Type)
	}
}
