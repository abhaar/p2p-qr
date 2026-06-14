// Package service defines broadcast service
package service

import (
	"context"
	"fmt"
	"log"

	"github.com/p2p/shared/pb/blockchain/broadcaster"
	"github.com/p2p/shared/pb/blockchain/network"
	"github.com/p2p/shared/pb/blockchain/protocol"
	signerpb "github.com/p2p/shared/pb/blockchain/signer"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
)

type BroadcastService struct {
	broadcaster.UnimplementedBroadcastServiceServer
	logger         *zap.Logger
	networkID      network.NetworkId
	protocolClient protocol.ProtocolServiceClient
	signer         signerpb.SigningServiceServer
}

func NewBroadcastService(logger *zap.Logger, networkID network.NetworkId, protocolClient protocol.ProtocolServiceClient, signer signerpb.SigningServiceServer) *BroadcastService {
	return &BroadcastService{
		logger:         logger,
		networkID:      networkID,
		protocolClient: protocolClient,
		signer:         signer,
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
		log.Printf("type: %+v", req.IntentPayload)
		return s.sendTransfer(ctx, req.GetIntentPayload())
	default:
		return nil, fmt.Errorf("invalid intent type: %T", req.IntentType)
	}
}

func (s *BroadcastService) sendTransfer(ctx context.Context, req *anypb.Any) (*broadcaster.TransactionIntentResponse, error) {
	s.logger.Info("received transfer request", zap.String("type", req.GetTypeUrl()))
	unsignedTx, err := s.protocolClient.PrepareTransaction(ctx, req)
	if err != nil {
		s.logger.Warn("prepare transfer failed", zap.Error(err))
		return nil, err
	}

	s.logger.Info("transfer request validated")

	signTransactionRequest := signerpb.UnsignedTransactionRequest{
		NetworkId: s.networkID,
		Request: &signerpb.UnsignedTransactionRequest_Evm{
			Evm: unsignedTx,
		},
	}

	s.logger.Info("getting transfer request signed")
	signedTransaction, err := s.signer.SignTransaction(ctx, &signTransactionRequest)
	if err != nil {
		s.logger.Warn("prepare transfer failed", zap.Error(err))
		return &broadcaster.TransactionIntentResponse{
			Status:       broadcaster.TransactionIntentResponse_STATUS_SIGNING_FAILED,
			ErrorMessage: err.Error(),
		}, nil
	}

	s.logger.Info("broadcasting transaction")
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

func (s *BroadcastService) GetTokenBalance(ctx context.Context, req *broadcaster.GetTokenBalanceRequest) (*broadcaster.GetTokenBalanceResponse, error) {
	if req == nil {
		return nil, fmt.Errorf("request must not be empty")
	}

	if req.NetworkId != s.networkID {
		return nil, fmt.Errorf("request received for invalid network: %s", req.NetworkId.String())
	}

	s.logger.Info("received get token balance request",
		zap.String("address", req.GetAddress()),
		zap.String("contract_address", req.GetContractAddress()),
	)

	resp, err := s.protocolClient.GetTokenBalance(ctx, &protocol.GetTokenBalanceRequest{
		Address:         req.GetAddress(),
		ContractAddress: req.GetContractAddress(),
	})
	if err != nil {
		s.logger.Warn("get token balance failed", zap.Error(err))
		return nil, err
	}

	return &broadcaster.GetTokenBalanceResponse{
		Balance: resp.GetBalance(),
	}, nil
}
