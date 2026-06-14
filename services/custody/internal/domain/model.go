// Package domain defines the core business models and interfaces for the custody service.
package domain

// Balance represents the token balance for an address.
type Balance struct {
	Address  string
	Currency string
	Amount   string // string to avoid floating-point precision issues
}

// TransferIntent represents a request to move tokens between two addresses.
type TransferIntent struct {
	From     string
	To       string
	Amount   string
	Currency string
}

// TransferResult represents the outcome of a submitted transfer.
type TransferResult struct {
	TxHash string
	Status string
}
