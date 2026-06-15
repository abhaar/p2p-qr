package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
	"github.com/skip2/go-qrcode"
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

	parsedAmount, err := parseDecimalAmount(req.Amount)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid amount: %v", err))
		return
	}

	result, err := s.custody.Transfer(r.Context(), domain.TransferIntent{
		From:     req.From,
		To:       req.To,
		Amount:   parsedAmount,
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

// parseDecimalAmount parses a decimal amount string, validates that it has at most 6 decimal places,
// and returns the amount multiplied by 1,000,000 as a big.Int string.
func parseDecimalAmount(amountStr string) (string, error) {
	amountStr = strings.TrimSpace(amountStr)
	if amountStr == "" {
		return "", fmt.Errorf("amount is empty")
	}

	d, err := decimal.NewFromString(amountStr)
	if err != nil {
		return "", fmt.Errorf("invalid decimal format")
	}

	if d.Sign() <= 0 {
		return "", fmt.Errorf("must be greater than zero")
	}

	// Multiply by 1,000,000
	multiplier := decimal.NewFromInt(1000000)
	scaled := d.Mul(multiplier)

	// If the scaled value is not an integer, it means the original number
	// had more than 6 decimal places.
	if !scaled.IsInteger() {
		return "", fmt.Errorf("exceeds maximum of 6 decimal places")
	}

	return scaled.StringFixed(0), nil
}

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

// handleGeneratePaymentQR generates a payment request QR code (GET/POST).
func (s *Server) handleGeneratePaymentQR(w http.ResponseWriter, r *http.Request) {
	var address, amount, currency, network, memo string

	if r.Method == http.MethodPost {
		var req struct {
			Address  string `json:"address"`
			Amount   string `json:"amount"`
			Currency string `json:"currency"`
			Network  string `json:"network"`
			Memo     string `json:"memo"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			s.writeError(w, http.StatusBadRequest, "invalid JSON body")
			return
		}
		address = req.Address
		amount = req.Amount
		currency = req.Currency
		network = req.Network
		memo = req.Memo
	} else if r.Method == http.MethodGet {
		address = r.URL.Query().Get("address")
		amount = r.URL.Query().Get("amount")
		currency = r.URL.Query().Get("currency")
		network = r.URL.Query().Get("network")
		memo = r.URL.Query().Get("memo")
	} else {
		s.writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	// Validate fields
	if address == "" || amount == "" || currency == "" || network == "" {
		s.writeError(w, http.StatusBadRequest, "address, amount, currency, and network are required")
		return
	}

	if !evmAddressRegex.MatchString(address) {
		s.writeError(w, http.StatusBadRequest, "invalid EVM address")
		return
	}

	dec, err := decimal.NewFromString(amount)
	if err != nil || dec.Sign() <= 0 {
		s.writeError(w, http.StatusBadRequest, "amount must be a valid positive decimal number")
		return
	}

	// Format amount back to a normalized string
	normalizedAmount := dec.String()

	payload := X9APaymentRequest{
		Address:  address,
		Amount:   normalizedAmount,
		Currency: currency,
		Network:  network,
		Memo:     memo,
	}

	qrContent, err := buildEMVCoQRContent(payload)
	if err != nil {
		s.logger.Error("failed to build EMVCo QR content", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "failed to generate QR content")
		return
	}

	// Generate QR code in PNG format
	png, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err != nil {
		s.logger.Error("failed to generate QR code image", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "failed to generate QR code")
		return
	}

	// Ensure qrcodes directory exists and save the file
	dir := "qrcodes"
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.logger.Error("failed to create qrcodes directory", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "failed to initialize storage")
		return
	}

	// Prepare base64-encoded image and HTML content
	base64Image := base64.StdEncoding.EncodeToString(png)
	htmlContent := strings.ReplaceAll(qrHTMLTemplate, "{{.QR_IMAGE_BASE64}}", base64Image)
	htmlContent = strings.ReplaceAll(htmlContent, "{{.PAYLOAD}}", qrContent)

	filename := fmt.Sprintf("qr_%s_%s.html", address, normalizedAmount)
	filePath := filepath.Join(dir, filename)
	if err := os.WriteFile(filePath, []byte(htmlContent), 0644); err != nil {
		s.logger.Error("failed to save QR HTML file", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "failed to save QR HTML file")
		return
	}

	// Return JSON or raw text based on Accept header
	if strings.Contains(r.Header.Get("Accept"), "application/json") {
		s.writeJSON(w, http.StatusOK, GenerateQRResponse{
			Payload: qrContent,
			QRImage: base64Image,
		})
	} else {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte(qrContent)); err != nil {
			s.logger.Error("failed to write raw payload response", zap.Error(err))
		}
	}
}

// handleTransferFromQR initiates a transfer using parameters parsed from a QR payload.
func (s *Server) handleTransferFromQR(w http.ResponseWriter, r *http.Request) {
	var req TransferFromQRRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.writeError(w, http.StatusBadRequest, "invalid JSON body")
		return
	}

	// Validate fields
	if req.Payload == "" || req.From == "" {
		s.writeError(w, http.StatusBadRequest, "payload and from are required")
		return
	}

	if !evmAddressRegex.MatchString(req.From) {
		s.writeError(w, http.StatusBadRequest, "invalid 'from' EVM address")
		return
	}

	// Parse payload
	parsed, err := parseEMVCoTLV(req.Payload)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("failed to parse QR code: %v", err))
		return
	}

	// Validate parsed data
	if parsed.Address == "" || parsed.Amount == "" || parsed.Currency == "" {
		s.writeError(w, http.StatusBadRequest, "missing address, amount, or currency in QR payload")
		return
	}

	if !evmAddressRegex.MatchString(parsed.Address) {
		s.writeError(w, http.StatusBadRequest, "invalid recipient address in QR payload")
		return
	}

	if !strings.EqualFold(parsed.Currency, "USDC") {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported currency in QR payload: %s", parsed.Currency))
		return
	}

	if !strings.EqualFold(parsed.Network, "ethereum") {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("unsupported network in QR payload: %s", parsed.Network))
		return
	}

	parsedAmount, err := parseDecimalAmount(parsed.Amount)
	if err != nil {
		s.writeError(w, http.StatusBadRequest, fmt.Sprintf("invalid amount in QR payload: %v", err))
		return
	}

	s.logger.Info("initiating QR-based transfer",
		zap.String("from", req.From),
		zap.String("to", parsed.Address),
		zap.String("amount", parsed.Amount),
		zap.String("currency", parsed.Currency),
		zap.String("memo", parsed.Memo),
	)

	result, err := s.custody.Transfer(r.Context(), domain.TransferIntent{
		From:     req.From,
		To:       parsed.Address,
		Amount:   parsedAmount,
		Currency: parsed.Currency,
	})
	if err != nil {
		s.logger.Error("QR-based transfer failed", zap.Error(err))
		s.writeError(w, http.StatusInternalServerError, "transfer failed")
		return
	}

	s.writeJSON(w, http.StatusOK, TransferResponse{
		TxHash: result.TxHash,
		Status: result.Status,
	})
}
