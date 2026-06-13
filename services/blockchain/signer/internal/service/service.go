// Package service defines the signer service.
package service

import (
	"context"
	"fmt"

	"github.com/p2p/blockchain/signer/v2/internal/evm"
	"github.com/p2p/shared/pb/blockchain/network"
	"github.com/p2p/shared/pb/blockchain/signer"
)

type InMemorySigner struct{}

func (s *InMemorySigner) SignTransaction(ctx context.Context, req *signer.UnsignedTransactionRequest) (*signer.SignedTransaction, error) {
	if req == nil {
		return nil, fmt.Errorf("request cannot be nil")
	}

	switch req.GetNetworkId() {
	case network.NetworkId_NETWORK_ID_ETHEREUM:
		evmUnsignedTransactionRequest := req.GetEvm()
		return evm.SignTransferIntent(evmUnsignedTransactionRequest)
	default:
		return nil, fmt.Errorf("network not supported: %s", req.GetNetworkId())
	}
}
