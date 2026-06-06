// Package domain contains internal models
package domain

import (
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

type EVMClient interface {
	RPCClient() *rpc.Client
}

type client struct {
	ethclient *ethclient.Client
}

func NewEVMClient(endpoint string) (EVMClient, error) {
	c, err := ethclient.Dial(endpoint)
	if err != nil {
		return &client{}, err
	}

	return &client{ethclient: c}, nil
}

func (c *client) RPCClient() *rpc.Client {
	return c.ethclient.Client()
}
