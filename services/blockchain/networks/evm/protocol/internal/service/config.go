package service

import (
	"fmt"
	"strconv"
	"strings"

	network "github.com/p2p/shared/pb/blockchain/network"
	"github.com/spf13/viper"
)

const (
	ServiceName = "blockchain-evm"
	defaultPort = ":50065"
)

type Config struct {
	GRPCServicePort        string
	NetworkID              network.NetworkId
	BlockchainNodeEndpoint string `mapstructure:"blockchain_node_endpoint"`
}

type rawConfig struct {
	GRPCServicePort        string
	NetworkID              string `mapstructure:"network_id"`
	BlockchainNodeEndpoint string `mapstructure:"blockchain_node_endpoint"`
}

func NewConfig() (*Config, error) {
	raw, err := LoadConfig[rawConfig]()
	if err != nil {
		return nil, err
	}

	netID, err := parseNetworkID(raw.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse network ID: %w", err)
	}

	return &Config{
		GRPCServicePort:        raw.GRPCServicePort,
		NetworkID:              netID,
		BlockchainNodeEndpoint: raw.BlockchainNodeEndpoint,
	}, nil
}

func parseNetworkID(id string) (network.NetworkId, error) {
	upper := strings.ToUpper(strings.TrimSpace(id))

	if val, ok := network.NetworkId_value[upper]; ok {
		return network.NetworkId(val), nil
	}

	if !strings.HasPrefix(upper, "NETWORK_ID_") {
		prefixed := "NETWORK_ID_" + upper
		if val, ok := network.NetworkId_value[prefixed]; ok {
			return network.NetworkId(val), nil
		}
	}

	if val, err := strconv.Atoi(upper); err == nil {
		if _, ok := network.NetworkId_name[int32(val)]; ok {
			return network.NetworkId(val), nil
		}
	}

	return network.NetworkId_NETWORK_ID_UNSPECIFIED, fmt.Errorf("unknown network ID: %s", id)
}

func LoadConfig[T any]() (*T, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &cfg, nil
}
