package client

import (
	"context"
	"fmt"

	"github.com/p2p/custody/v2/internal/domain"
	"github.com/p2p/shared/pb/blockchain/address"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type AddressClient struct {
	conn   *grpc.ClientConn
	client address.AddressServiceClient
}

var _ domain.AddressService = (*AddressClient)(nil)

func NewAddressClient(addr string) (*AddressClient, error) {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, fmt.Errorf("address service grpc dial: %w", err)
	}
	return &AddressClient{
		conn:   conn,
		client: address.NewAddressServiceClient(conn),
	}, nil
}

func (c *AddressClient) GetCustodyAddresses(ctx context.Context, addresses []string) ([]string, error) {
	resp, err := c.client.GetCustodyAddresses(ctx, &address.GetCustodyAddressesRequest{
		Addresses: addresses,
	})
	if err != nil {
		return nil, fmt.Errorf("address service GetCustodyAddresses: %w", err)
	}
	return resp.GetAddresses(), nil
}

func (c *AddressClient) Close() error {
	return c.conn.Close()
}
