// Package api contains the interface for custody service
package api

// TransferRequest is the JSON body accepted by POST /transfer.
type TransferRequest struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

// --- Response DTOs ---

// BalanceResponse is returned by the GET /balance endpoint.
type BalanceResponse struct {
	Address  string `json:"address"`
	Currency string `json:"currency"`
	Balance  string `json:"balance"`
}

// TransferResponse is returned by the POST /transfer endpoint.
type TransferResponse struct {
	TxHash string `json:"tx_hash"`
	Status string `json:"status"`
}

// ErrorResponse is returned when an API request fails.
type ErrorResponse struct {
	Error string `json:"error"`
}
