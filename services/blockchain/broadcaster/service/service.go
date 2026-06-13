// Package service defines broadcast service
package service

import (
	"context"
	"fmt"

	"github.com/p2p/blockchain/signer/v2/api"
	"github.com/p2p/shared/pb/blockchain/broadcaster"
	"github.com/p2p/shared/pb/blockchain/network"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"github.com/p2p/shared/pb/blockchain/signer"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
)

type BroadcastService struct {
	broadcaster.UnimplementedBroadcastServiceServer
	logger         *zap.Logger
	networkID      network.NetworkId
	protocolClient protocol.ProtocolServiceClient
	signer         api.Signer
}

func NewBroadcastService(logger *zap.Logger, networkId network.NetworkId, protocolClient protocol.ProtocolServiceClient) *BroadcastService {
	return &BroadcastService{
		logger:         logger,
		networkID:      networkId,
		protocolClient: protocolClient,
	}
}

func (s *BroadcastService) SendTransaction(ctx context.Context, req *broadcaster.TransactionIntentRequest) (*broadcaster.TransactionIntentResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be empty")
	}

	if req.NetworkId != s.networkID {
		return nil, fmt.Errorf("request received for invalid network: %s", req.NetworkId.String())
	}

	switch req.IntentType {
	case broadcaster.TransactionIntentRequest_INTENT_TYPE_TRANSFER:
		return s.sendTransfer(ctx, req.GetIntentPayload())
	default:
		return nil, fmt.Errorf("invalid intent type: %T", req.IntentType)
	}
}

func (s *BroadcastService) sendTransfer(ctx context.Context, req *anypb.Any) (*broadcaster.TransactionIntentResponse, error) {
	unsignedTx, err := s.protocolClient.PrepareTransaction(ctx, req)
	if err != nil {
		s.logger.Warn("prepare transfer failed", zap.Error(err))
		return nil, err
	}

	signTransactionRequest := signer.UnsignedTransactionRequest{
		NetworkId: s.networkID,
		Request: &signer.UnsignedTransactionRequest_Evm{
			Evm: unsignedTx,
		},
	}

	signedTransaction, err := s.signer.SignTransaction(ctx, &signTransactionRequest)
	if err != nil {
		s.logger.Warn("prepare transfer failed", zap.Error(err))
		return &broadcaster.TransactionIntentResponse{
			Status:       broadcaster.TransactionIntentResponse_STATUS_SIGNING_FAILED,
			ErrorMessage: err.Error(),
		}, nil
	}

	txResponse, err := s.protocolClient.Transfer(ctx, signedTransaction)
	if err != nil {
		s.logger.Error("transfer failed", zap.Error(err))
		return &broadcaster.TransactionIntentResponse{
			Status:       broadcaster.TransactionIntentResponse_STATUS_BROADCAST_FAILED,
			ErrorMessage: err.Error(),
		}, nil
	}

	if txResponse.GetStatus() == protocol.TransferResponse_TRANSFER_RESPONSE_FAILED {
		s.logger.Error("transfer failed", zap.Stringer("error", txResponse))
		return &broadcaster.TransactionIntentResponse{
			Status:       broadcaster.TransactionIntentResponse_STATUS_BROADCAST_FAILED,
			ErrorMessage: txResponse.GetError(),
		}, nil
	}

	return &broadcaster.TransactionIntentResponse{
		Status:        broadcaster.TransactionIntentResponse_STATUS_BROADCASTED,
		TransactionId: txResponse.GetTxHash(),
	}, nil
}
