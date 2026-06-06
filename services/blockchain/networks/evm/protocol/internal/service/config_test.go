package service

import (
	"testing"

	network "github.com/p2p/shared/pb/blockchain/network"
)

func TestParseNetworkID(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    network.NetworkId
		wantErr bool
	}{
		{
			name:    "Ethereum name direct",
			input:   "NETWORK_ID_ETHEREUM",
			want:    network.NetworkId_NETWORK_ID_ETHEREUM,
			wantErr: false,
		},
		{
			name:    "Ethereum name simple",
			input:   "ETHEREUM",
			want:    network.NetworkId_NETWORK_ID_ETHEREUM,
			wantErr: false,
		},
		{
			name:    "Ethereum name lowercase",
			input:   "ethereum",
			want:    network.NetworkId_NETWORK_ID_ETHEREUM,
			wantErr: false,
		},
		{
			name:    "Ethereum name lowercase prefix",
			input:   "network_id_ethereum",
			want:    network.NetworkId_NETWORK_ID_ETHEREUM,
			wantErr: false,
		},
		{
			name:    "Ethereum numeric",
			input:   "1",
			want:    network.NetworkId_NETWORK_ID_ETHEREUM,
			wantErr: false,
		},
		{
			name:    "Unspecified name direct",
			input:   "NETWORK_ID_UNSPECIFIED",
			want:    network.NetworkId_NETWORK_ID_UNSPECIFIED,
			wantErr: false,
		},
		{
			name:    "Unspecified name simple",
			input:   "UNSPECIFIED",
			want:    network.NetworkId_NETWORK_ID_UNSPECIFIED,
			wantErr: false,
		},
		{
			name:    "Unspecified numeric",
			input:   "0",
			want:    network.NetworkId_NETWORK_ID_UNSPECIFIED,
			wantErr: false,
		},
		{
			name:    "Invalid name",
			input:   "SOLANA",
			want:    network.NetworkId_NETWORK_ID_UNSPECIFIED,
			wantErr: true,
		},
		{
			name:    "Invalid numeric",
			input:   "99",
			want:    network.NetworkId_NETWORK_ID_UNSPECIFIED,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseNetworkID(tt.input)
			if (err != nil) != tt.wantErr {
				t.Fatalf("parseNetworkID() error = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Errorf("parseNetworkID() = %v, want %v", got, tt.want)
			}
		})
	}
}
