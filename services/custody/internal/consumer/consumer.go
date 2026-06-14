// Package consumer
package consumer

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"strings"

	"github.com/nats-io/nats.go"
	"github.com/p2p/custody/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

type EventConsumer struct {
	logger               *zap.Logger
	addressServiceClient domain.AddressService
	nc                   *nats.Conn
	sub                  *nats.Subscription
	repo                 domain.Repository
	subject              string
}

func getUSDCContractAddress() string {
	if envAddr := os.Getenv("USDC_CONTRACT_ADDRESS"); envAddr != "" {
		return envAddr
	}
	return "0x5FbDB2315678afecb367f032d93F642f64180aa3"
}

func NewEventConsumer(logger *zap.Logger, addClient domain.AddressService, natsURL string, subject string, repo domain.Repository) (*EventConsumer, error) {
	nc, err := nats.Connect(natsURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS at %s: %w", natsURL, err)
	}

	return &EventConsumer{
		logger:               logger,
		addressServiceClient: addClient,
		nc:                   nc,
		repo:                 repo,
		subject:              subject,
	}, nil
}

func (c *EventConsumer) Start(ctx context.Context) error {
	c.logger.Info("starting NATS event consumer", zap.String("subject", c.subject))

	sub, err := c.nc.Subscribe(c.subject, func(msg *nats.Msg) {
		var events protocol.BlockchainEvents
		if err := proto.Unmarshal(msg.Data, &events); err != nil {
			c.logger.Error("failed to unmarshal blockchain events", zap.Error(err))
			return
		}

		for _, event := range events.GetEvents() {
			c.processEvent(ctx, event)
		}
	})
	if err != nil {
		return fmt.Errorf("failed to subscribe to NATS subject %s: %w", c.subject, err)
	}
	c.sub = sub

	<-ctx.Done()
	return nil
}

func (c *EventConsumer) processEvent(ctx context.Context, event *protocol.BlockchainEvent) {
	tokenAddress := strings.ToLower(event.GetTokenId())
	usdcAddress := strings.ToLower(getUSDCContractAddress())

	if tokenAddress != usdcAddress {
		return
	}

	fromAddr := event.GetFrom()
	toAddr := event.GetTo()
	amountStr := event.GetAmount()

	amount, ok := new(big.Int).SetString(amountStr, 10)
	if !ok {
		c.logger.Error("failed to parse event amount", zap.String("amount", amountStr))
		return
	}

	custodiedAddress, err := c.addressServiceClient.GetCustodyAddresses(ctx, []string{fromAddr, toAddr})
	if err != nil {
		c.logger.Error("failed to get custodied addresses from address service", zap.Error(err))
	}

	if len(custodiedAddress) > 0 {
		c.logger.Info("processing USDC token transfer",
			zap.String("from", fromAddr),
			zap.String("to", toAddr),
			zap.String("amount", amount.String()),
			zap.String("tx_hash", event.GetTxHash()),
		)
	}

	custodiedAddressSet := make(map[string]struct{})
	for _, addr := range custodiedAddress {
		custodiedAddressSet[addr] = struct{}{}
	}

	if _, ok := custodiedAddressSet[fromAddr]; ok {
		negAmount := new(big.Int).Mul(amount, big.NewInt(-1))
		err = c.repo.UpdateBalance(negAmount, fromAddr, "USDC")
		if err != nil {
			c.logger.Error("failed to update balance", zap.Error(err))
		}
	}

	if _, ok := custodiedAddressSet[toAddr]; ok {
		err = c.repo.UpdateBalance(amount, toAddr, "USDC")
		if err != nil {
			c.logger.Error("failed to update balance", zap.Error(err))
		}
	}
}

func (c *EventConsumer) Close() {
	if c.sub != nil {
		c.sub.Unsubscribe()
	}
	if c.nc != nil {
		c.nc.Close()
	}
}
