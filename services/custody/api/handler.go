package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"go.uber.org/zap"

	"github.com/p2p/custody/v2/internal/domain"
)

// evmAddressRegex validates EVM-compatible hex addresses (0x followed by 40 hex chars).
var evmAddressRegex = regexp.MustCompile(`^0x[0-9a-fA-F]{40}$`)

// handleGetBalance returns the USDC balance for the supplied EVM address.
func (s *Server) handleGetBalance(w http.ResponseWriter, r *http.Request) {
	address := r.PathValue("address")
	if address == "" {
		s.writeError(w, http.StatusBadRequest, "address is required")
		return
	}

	address = strings.ToLower(address)
	if !evmAddressRegex.MatchString(address) {
		s.writeError(w, http.StatusBadRequest, "invalid EVM address")
		return
	}

	balance, err := s.custody.GetBalance(r.Context(), address, "USDC")
	if err != nil {
		s.logger.Error("failed to get balance", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "failed to retrieve balance")
		return
	}

	s.writeJSON(w, http.StatusOK, BalanceResponse{
		Address:  balance.Address,
		Currency: balance.Currency,
		Balance:  balance.Amount,
	})
}

// handleTransfer initiates a USDC transfer between two EVM addresses.
func (s *Server) handleTransfer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// --- Validate fields ---
	if req.From == "" || req.To == "" || req.Amount == "" || req.Currency == "" {
		s.writeError(w, http.StatusBadRequest, "from, to, amount, and currency are required")
		return
	}

	req.From = strings.ToLower(req.From)
	req.To = strings.ToLower(req.To)

	if !evmAddressRegex.MatchString(req.From) {
		s.writeError(w, http.StatusBadRequest, "invalid 'from' EVM address")
		return
	}
	if !evmAddressRegex.MatchString(req.To) {
		s.writeError(w, http.StatusBadRequest, "invalid 'to' EVM address")
		return
	}

	if !strings.EqualFold(req.Currency, "USDC") {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported currency: %s", req.Currency))
		return
	}

	result, err := s.custody.Transfer(r.Context(), domain.TransferIntent{
		From:     req.From,
		To:       req.To,
		Amount:   req.Amount,
		Currency: req.Currency,
	})
	if err != nil {
		s.logger.Error("transfer failed", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "transfer failed")
		return
	}

	s.writeJSON(w, http.StatusOK, TransferResponse{
		TxHash: result.TxHash,
		Status: result.Status,
	})
}

// --- Helpers ---

// writeJSON marshals v as JSON and writes it to w with the given status code.
func (s *Server) writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		s.logger.Error("failed to encode response", zap.Error(err))
	}
}

// writeError writes a JSON error response.
func (s *Server) writeError(w http.ResponseWriter, status int, msg string) {
	s.writeJSON(w, status, ErrorResponse{Error: msg})
}
