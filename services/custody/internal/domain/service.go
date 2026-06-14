package domain

import "context"

// CustodyService defines the business operations the custody service supports.
// The API layer depends on this interface, not on a concrete implementation.
type CustodyService interface {
	// GetBalance returns the token balance for the given address and currency.
	GetBalance(ctx context.Context, address, currency string) (*Balance, error)

	// Transfer executes a token transfer described by the given intent.
	Transfer(ctx context.Context, intent TransferIntent) (*TransferResult, error)
}

// Broadcaster abstracts the blockchain broadcaster service.
// The concrete implementation wraps the gRPC BroadcastServiceClient.
type Broadcaster interface {
	// SendTransfer submits an ERC-20 transfer to the broadcaster service and
	// returns the resulting transaction hash and status.
	SendTransfer(ctx context.Context, intent TransferIntent) (*TransferResult, error)
}
