// Package publisher
package publisher

import (
	"context"

	"github.com/p2p/shared/pb/blockchain/protocol"
	"go.uber.org/zap"
)

type EventPublisher interface {
	Publish(ctx context.Context, logger *zap.Logger, events *protocol.BlockchainEvents) error
}
