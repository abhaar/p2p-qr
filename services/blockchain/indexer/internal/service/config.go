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
	ServiceName = "blockchain-indexer"
	defaultPort = ":50066"
)

type Config struct {
	NetworkID               network.NetworkId
	ProtocolServiceEndpoint string
	StartingBlock           uint64
	NatsURL                 string
	NatsSubject             string
}

type rawConfig struct {
	NetworkID               string `mapstructure:"network_id"`
	ProtocolServiceEndpoint string `mapstructure:"protocol_service_endpoint"`
	StartingBlock           uint64 `mapstructure:"starting_block"`
	NatsURL                 string `mapstructure:"nats_url"`
	NatsSubject             string `mapstructure:"nats_subject"`
}

func NewConfig() (*Config, error) {
	var (
		flagNetwork          = flag.String("network-id", "", "Network ID")
		flagProtocolEndpoint = flag.String("protocol-service-endpoint", "", "Protocol service endpoint")
		flagStartingBlock    = flag.Uint64("starting-block", 0, "Starting block height")
		flagNatsURL          = flag.String("nats-url", "", "NATS URL")
		flagNatsSubject      = flag.String("nats-subject", "", "NATS Subject")
	)

	if !flag.Parsed() {
		flag.Parse()
	}

	var startingBlockSet bool
	var natsURLSet bool
	var natsSubjectSet bool
	flag.Visit(func(f *flag.Flag) {
		switch f.Name {
		case "starting-block":
			startingBlockSet = true
		case "nats-url":
			natsURLSet = true
		case "nats-subject":
			natsSubjectSet = true
		}
	})

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
	if *flagNetwork != "" {
		raw.NetworkID = *flagNetwork
	}
	if *flagProtocolEndpoint != "" {
		raw.ProtocolServiceEndpoint = *flagProtocolEndpoint
	}
	if startingBlockSet {
		raw.StartingBlock = *flagStartingBlock
	}
	if natsURLSet {
		raw.NatsURL = *flagNatsURL
	}
	if natsSubjectSet {
		raw.NatsSubject = *flagNatsSubject
	}

	// Apply defaults
	if raw.NatsURL == "" {
		raw.NatsURL = "nats://localhost:4222"
	}
	if raw.NatsSubject == "" {
		raw.NatsSubject = "blockchain.events"
	}

	// If we still don't have a network ID or blockchain node endpoint, return an error
	if raw.NetworkID == "" {
		if configFileNotFound {
			return nil, fmt.Errorf("network ID must be provided via config file or -network-id flag")
		}
		return nil, fmt.Errorf("network ID is empty")
	}

	netID, err := parseNetworkID(raw.NetworkID)
	if err != nil {
		return nil, fmt.Errorf("failed to parse network ID: %w", err)
	}

	return &Config{
		NetworkID:               netID,
		ProtocolServiceEndpoint: raw.ProtocolServiceEndpoint,
		StartingBlock:           raw.StartingBlock,
		NatsURL:                 raw.NatsURL,
		NatsSubject:             raw.NatsSubject,
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
