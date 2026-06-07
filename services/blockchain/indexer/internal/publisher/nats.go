package publisher

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/p2p/shared/pb/blockchain/protocol"
	"google.golang.org/protobuf/proto"
)

type NatsPublisher struct {
	nc      *nats.Conn
	subject string
}

func NewNatsPublisher(url string, subject string) (*NatsPublisher, error) {
	nc, err := nats.Connect(url)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	return &NatsPublisher{
		nc:      nc,
		subject: subject,
	}, nil
}

func (p *NatsPublisher) Publish(ctx context.Context, events *protocol.BlockchainEvents) error {
	if len(events.GetEvents()) == 0 {
		return nil
	}

	data, err := proto.Marshal(events)
	if err != nil {
		return fmt.Errorf("failed to marshal blockchain events: %w", err)
	}

	err = p.nc.Publish(p.subject, data)
	if err != nil {
		return fmt.Errorf("failed to publish message to NATS: %w", err)
	}

	return nil
}

func (p *NatsPublisher) Close() {
	if p.nc != nil {
		p.nc.Close()
	}
}
