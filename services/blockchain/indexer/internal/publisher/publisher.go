// Package publisher
package publisher

import (
	"context"

	"github.com/p2p/shared/pb/blockchain/protocol"
)

type EventPublisher interface {
	Publish(ctx context.Context, events *protocol.BlockchainEvents) error
}
