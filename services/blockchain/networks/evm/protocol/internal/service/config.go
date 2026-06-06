package service

import (
	"errors"
	"flag"
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
	AddressServiceEndpoint string
}

type rawConfig struct {
	GRPCServicePort        string
	NetworkID              string `mapstructure:"network_id"`
	BlockchainNodeEndpoint string `mapstructure:"blockchain_node_endpoint"`
	AddressServiceEndpoint string `mapstructure:"address_service_endpoint"`
}

func NewConfig() (*Config, error) {
	var (
		flagPort            = flag.String("grpc-port", "", "gRPC service port")
		flagNetwork         = flag.String("network-id", "", "Network ID")
		flagEndpoint        = flag.String("node-endpoint", "", "Blockchain node endpoint")
		flagAddressEndpoint = flag.String("address-service-endpoint", "", "Address service endpoint")
	)

	if !flag.Parsed() {
		flag.Parse()
	}

	raw, err := LoadConfig[rawConfig]()
	var configFileNotFound bool
	if err != nil {
		var configFileNotFoundErr viper.ConfigFileNotFoundError
		if errors.As(err, &configFileNotFoundErr) {
			configFileNotFound = true
			raw = &rawConfig{}
		} else {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Override with flags if they are set
	if *flagPort != "" {
		raw.GRPCServicePort = *flagPort
	}
	if *flagNetwork != "" {
		raw.NetworkID = *flagNetwork
	}
	if *flagEndpoint != "" {
		raw.BlockchainNodeEndpoint = *flagEndpoint
	}
	if *flagAddressEndpoint != "" {
		raw.AddressServiceEndpoint = *flagAddressEndpoint
	}

	// Fallback to default port if not set anywhere
	if raw.GRPCServicePort == "" {
		raw.GRPCServicePort = defaultPort
	}

	// Fallback to default address service endpoint if not set
	if raw.AddressServiceEndpoint == "" {
		raw.AddressServiceEndpoint = "localhost:50064"
	}

	// If we still don't have a network ID or blockchain node endpoint, return an error
	if raw.NetworkID == "" {
		if configFileNotFound {
			return nil, fmt.Errorf("network ID must be provided via config file or -network-id flag")
		}
		return nil, fmt.Errorf("network ID is empty")
	}
	if raw.BlockchainNodeEndpoint == "" {
		if configFileNotFound {
			return nil, fmt.Errorf("blockchain node endpoint must be provided via config file or -node-endpoint flag")
		}
		return nil, fmt.Errorf("blockchain node endpoint is empty")
	}

	netID, err := parseNetworkID(raw.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse network ID: %w", err)
	}

	return &Config{
		GRPCServicePort:        raw.GRPCServicePort,
		NetworkID:              netID,
		BlockchainNodeEndpoint: raw.BlockchainNodeEndpoint,
		AddressServiceEndpoint: raw.AddressServiceEndpoint,
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
		return nil, err
	}

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	return &cfg, nil
}
