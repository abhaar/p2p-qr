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

	// Return raw payload string in response
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte(qrContent)); err != nil {
		s.logger.Error("failed to write raw payload response", zap.Error(err))
	}
}

// qrHTMLTemplate is the template for self-contained, interactive payment QR request pages.
const qrHTMLTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>X9.150 Payment QR Code Viewer</title>
    <!-- Google Fonts Outfit -->
    <link rel="preconnect" href="https://fonts.googleapis.com">
    <link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
    <link href="https://fonts.googleapis.com/css2?family=Outfit:wght@300;400;600;800&display=swap" rel="stylesheet">
    <style>
        :root {
            --bg-color: #0b0f19;
            --card-bg: rgba(17, 24, 39, 0.7);
            --accent-color: #3b82f6;
            --accent-glow: rgba(59, 130, 246, 0.4);
            --text-primary: #f3f4f6;
            --text-secondary: #9ca3af;
            --success-color: #10b981;
        }

        * {
            box-sizing: border-box;
            margin: 0;
            padding: 0;
        }

        body {
            font-family: 'Outfit', sans-serif;
            background: radial-gradient(circle at 50% 50%, #1e293b 0%, var(--bg-color) 100%);
            color: var(--text-primary);
            min-height: 100vh;
            display: flex;
            align-items: center;
            justify-content: center;
            overflow: hidden;
            position: relative;
        }

        /* Ambient background glow */
        body::before {
            content: '';
            position: absolute;
            width: 300px;
            height: 300px;
            background: var(--accent-glow);
            filter: blur(120px);
            border-radius: 50%;
            top: 20%;
            left: 30%;
            z-index: 0;
        }

        body::after {
            content: '';
            position: absolute;
            width: 250px;
            height: 250px;
            background: rgba(16, 185, 129, 0.2);
            filter: blur(100px);
            border-radius: 50%;
            bottom: 20%;
            right: 30%;
            z-index: 0;
        }

        .container {
            position: relative;
            z-index: 10;
            background: var(--card-bg);
            backdrop-filter: blur(16px);
            -webkit-backdrop-filter: blur(16px);
            border: 1px solid rgba(255, 255, 255, 0.08);
            border-radius: 24px;
            padding: 40px;
            width: 100%;
            max-width: 440px;
            text-align: center;
            box-shadow: 0 20px 40px rgba(0, 0, 0, 0.3);
            transition: transform 0.3s ease, box-shadow 0.3s ease;
        }

        .container:hover {
            transform: translateY(-4px);
            box-shadow: 0 24px 48px rgba(0, 0, 0, 0.4), 0 0 30px var(--accent-glow);
            border-color: rgba(59, 130, 246, 0.2);
        }

        h1 {
            font-size: 24px;
            font-weight: 800;
            margin-bottom: 8px;
            background: linear-gradient(135deg, #60a5fa 0%, #34d399 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            letter-spacing: -0.5px;
        }

        p.subtitle {
            font-size: 14px;
            color: var(--text-secondary);
            margin-bottom: 30px;
        }

        .qr-wrapper {
            position: relative;
            background: rgba(255, 255, 255, 0.03);
            border: 1px dashed rgba(255, 255, 255, 0.15);
            border-radius: 16px;
            padding: 24px;
            display: inline-block;
            cursor: pointer;
            transition: all 0.3s ease;
            margin-bottom: 24px;
        }

        .qr-wrapper:hover {
            background: rgba(59, 130, 246, 0.05);
            border-color: var(--accent-color);
            transform: scale(1.02);
        }

        .qr-image {
            display: block;
            width: 200px;
            height: 200px;
            border-radius: 8px;
            pointer-events: none;
        }

        .hover-overlay {
            position: absolute;
            top: 0;
            left: 0;
            width: 100%;
            height: 100%;
            background: rgba(0, 0, 0, 0.5);
            border-radius: 16px;
            opacity: 0;
            display: flex;
            align-items: center;
            justify-content: center;
            transition: opacity 0.3s ease;
            color: white;
            font-size: 14px;
            font-weight: 600;
        }

        .qr-wrapper:hover .hover-overlay {
            opacity: 1;
        }

        .btn-copy {
            background: linear-gradient(135deg, #2563eb 0%, #1d4ed8 100%);
            color: white;
            border: none;
            padding: 12px 24px;
            font-family: inherit;
            font-size: 15px;
            font-weight: 600;
            border-radius: 12px;
            cursor: pointer;
            transition: all 0.2s ease;
            box-shadow: 0 4px 12px rgba(37, 99, 235, 0.3);
            width: 100%;
            margin-bottom: 20px;
        }

        .btn-copy:hover {
            transform: translateY(-1px);
            box-shadow: 0 6px 16px rgba(37, 99, 235, 0.4);
            filter: brightness(1.1);
        }

        .btn-copy:active {
            transform: translateY(1px);
        }

        .payload-preview {
            background: rgba(0, 0, 0, 0.2);
            border: 1px solid rgba(255, 255, 255, 0.05);
            border-radius: 10px;
            padding: 12px;
            font-family: monospace;
            font-size: 11px;
            color: var(--text-secondary);
            word-break: break-all;
            max-height: 80px;
            overflow-y: auto;
            text-align: left;
        }

        /* Toast notification */
        .toast {
            position: fixed;
            bottom: -60px;
            left: 50%;
            transform: translateX(-50%);
            background: var(--success-color);
            color: white;
            padding: 12px 24px;
            border-radius: 30px;
            font-weight: 600;
            font-size: 14px;
            box-shadow: 0 10px 20px rgba(16, 185, 129, 0.3);
            display: flex;
            align-items: center;
            gap: 8px;
            transition: bottom 0.4s cubic-bezier(0.175, 0.885, 0.32, 1.275);
            z-index: 100;
        }

        .toast.show {
            bottom: 40px;
        }
    </style>
</head>
<body>

    <div class="container">
        <h1>X9.150 QR Payload</h1>
        <p class="subtitle">Click the QR code or button to copy raw TLV payload</p>

        <div class="qr-wrapper" id="qrWrapper">
            <img class="qr-image" src="data:image/png;base64,{{.QR_IMAGE_BASE64}}" alt="X9.150 Payment QR Code">
            <div class="hover-overlay">
                Click to Copy Payload
            </div>
        </div>

        <button class="btn-copy" id="btnCopy">Copy Raw Payload</button>

        <div class="payload-preview" id="payloadText">{{.PAYLOAD}}</div>
    </div>

    <div class="toast" id="toast">
        <svg width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="3" stroke-linecap="round" stroke-linejoin="round"><polyline points="20 6 9 17 4 12"></polyline></svg>
        Copied to clipboard!
    </div>

    <script>
        const payload = "{{.PAYLOAD}}";
        
        const copyToClipboard = () => {
            navigator.clipboard.writeText(payload).then(() => {
                showToast();
            }).catch(err => {
                console.error("Failed to copy: ", err);
            });
        };

        const showToast = () => {
            const toast = document.getElementById("toast");
            toast.classList.add("show");
            setTimeout(() => {
                toast.classList.remove("show");
            }, 2000);
        };

        document.getElementById("qrWrapper").addEventListener("click", copyToClipboard);
        document.getElementById("btnCopy").addEventListener("click", copyToClipboard);
    </script>
</body>
</html>`

// formatEMVCoTLV formats a Tag-Length-Value field.
func formatEMVCoTLV(tag string, value string) string {
	return fmt.Sprintf("%s%02d%s", tag, len(value), value)
}

// calculateCRC16 calculates the CRC-16/CCITT-FALSE checksum for EMVCo specifications.
func calculateCRC16(data string) string {
	crc := uint16(0xFFFF)
	polynomial := uint16(0x1021)
	for i := 0; i < len(data); i++ {
		crc ^= uint16(data[i]) << 8
		for j := 0; j < 8; j++ {
			if (crc & 0x8000) != 0 {
				crc = (crc << 1) ^ polynomial
			} else {
				crc <<= 1
			}
		}
	}
	return fmt.Sprintf("%04X", crc)
}

// buildEMVCoQRContent builds the X9.150 compliant EMVCo MPM QR payload.
func buildEMVCoQRContent(req X9APaymentRequest) (string, error) {
	var parts []string

	// Tag 00: Payload Format Indicator (mandatory, value = "01")
	parts = append(parts, formatEMVCoTLV("00", "01"))

	// Tag 01: Point of Initiation Method (mandatory, dynamic = "12")
	parts = append(parts, formatEMVCoTLV("01", "12"))

	// Tag 26: Merchant Account Information (mandatory for X9.150)
	var t26Parts []string
	// Sub-tag 00: Globally Unique Identifier (GUI)
	t26Parts = append(t26Parts, formatEMVCoTLV("00", "org.x9.payment"))
	if req.Address != "" {
		t26Parts = append(t26Parts, formatEMVCoTLV("01", req.Address))
	}
	if req.Network != "" {
		t26Parts = append(t26Parts, formatEMVCoTLV("02", req.Network))
	}
	if req.Currency != "" {
		t26Parts = append(t26Parts, formatEMVCoTLV("03", req.Currency))
	}
	t26Value := strings.Join(t26Parts, "")
	parts = append(parts, formatEMVCoTLV("26", t26Value))

	// Tag 52: Merchant Category Code (standard placeholder)
	parts = append(parts, formatEMVCoTLV("52", "0000"))

	// Tag 53: Transaction Currency (ISO 4217, USD = "840")
	parts = append(parts, formatEMVCoTLV("53", "840"))

	// Tag 54: Transaction Amount
	parts = append(parts, formatEMVCoTLV("54", req.Amount))

	// Tag 58: Country Code (ISO 3166)
	parts = append(parts, formatEMVCoTLV("58", "US"))

	// Tag 59: Merchant Name
	parts = append(parts, formatEMVCoTLV("59", "Custody API"))

	// Tag 60: Merchant City
	parts = append(parts, formatEMVCoTLV("60", "New York"))

	// Tag 62: Additional Data Template (optional, for memo)
	if req.Memo != "" {
		var t62Parts []string
		t62Parts = append(t62Parts, formatEMVCoTLV("05", req.Memo))
		t62Value := strings.Join(t62Parts, "")
		parts = append(parts, formatEMVCoTLV("62", t62Value))
	}

	// Tag 63: CRC16 checksum
	payloadStrWithoutCRC := strings.Join(parts, "") + "6304"
	crc := calculateCRC16(payloadStrWithoutCRC)
	return payloadStrWithoutCRC + crc, nil
}
