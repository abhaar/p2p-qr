package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"

	"github.com/p2p/custody/v2/internal/domain"
)

func TestParseDecimalAmount(t *testing.T) {
	tests := []struct {
		input    string
		expected string
		wantErr  bool
	}{
		{"12.345", "12345000", false},
		{"0.000001", "1", false},
		{"10", "10000000", false},
		{"0", "", true}, // Must be greater than zero
		{"-5.5", "", true},
		{"12.3456789", "", true},
		{"12a.34", "", true},
		{"", "", true},
		{".5", "500000", false},
		{"5.", "5000000", false},
		{" 1.5 ", "1500000", false},
		{"1.5000000000", "1500000", false}, // Trailing zeros should be normalized out
		{"1.0000000001", "", true},       // Exceeds 6 decimal places
		{"1e-6", "1", false},              // Scientific notation (1 * 10^-6)
	}

	for _, tt := range tests {
		got, err := parseDecimalAmount(tt.input)
		if (err != nil) != tt.wantErr {
			t.Errorf("parseDecimalAmount(%q) error = %v, wantErr = %v", tt.input, err, tt.wantErr)
			continue
		}
		if !tt.wantErr && got != tt.expected {
			t.Errorf("parseDecimalAmount(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestHandleGeneratePaymentQR(t *testing.T) {
	// Clean up any generated files in qrcodes folder
	defer os.RemoveAll("qrcodes")

	logger := zap.NewNop()
	server := NewServer(logger, nil)

	t.Run("GET - valid request", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/payment-request/qr?address=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266&amount=10.50&currency=USDC&network=ethereum&memo=test-payment", nil)
		w := httptest.NewRecorder()
		server.handleGeneratePaymentQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status OK, got %v", resp.Status)
		}
		if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/plain") {
			t.Errorf("expected Content-Type text/plain, got %v", contentType)
		}

		bodyBytes := w.Body.Bytes()
		payloadStr := string(bodyBytes)
		if payloadStr == "" {
			t.Errorf("expected non-empty response payload string")
		}


	})

	t.Run("GET - missing parameters", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/payment-request/qr?address=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266&amount=10.50", nil)
		w := httptest.NewRecorder()
		server.handleGeneratePaymentQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %v", resp.Status)
		}
	})

	t.Run("GET - invalid EVM address", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/payment-request/qr?address=invalid-address&amount=10.50&currency=USDC&network=ethereum", nil)
		w := httptest.NewRecorder()
		server.handleGeneratePaymentQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %v", resp.Status)
		}
	})

	t.Run("GET - invalid amount", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/payment-request/qr?address=0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266&amount=-5.00&currency=USDC&network=ethereum", nil)
		w := httptest.NewRecorder()
		server.handleGeneratePaymentQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %v", resp.Status)
		}
	})

	t.Run("POST - valid request", func(t *testing.T) {
		body := `{"address":"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266","amount":"123.45","currency":"USDC","network":"ethereum","memo":"test-post"}`
		req := httptest.NewRequest("POST", "/payment-request/qr", strings.NewReader(body))
		w := httptest.NewRecorder()
		server.handleGeneratePaymentQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status OK, got %v", resp.Status)
		}
		if contentType := resp.Header.Get("Content-Type"); !strings.Contains(contentType, "text/plain") {
			t.Errorf("expected Content-Type text/plain, got %v", contentType)
		}

		bodyBytes := w.Body.Bytes()
		payloadStr := string(bodyBytes)
		if payloadStr == "" {
			t.Errorf("expected non-empty response payload string")
		}


	})

	t.Run("POST - invalid JSON body", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/payment-request/qr", strings.NewReader(`{invalid`))
		w := httptest.NewRecorder()
		server.handleGeneratePaymentQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %v", resp.Status)
		}
	})
}

type MockCustodyService struct {
	TransferFunc func(ctx context.Context, intent domain.TransferIntent) (*domain.TransferResult, error)
}

func (m *MockCustodyService) GetBalance(ctx context.Context, address, currency string) (domain.Balance, error) {
	return domain.Balance{}, nil
}

func (m *MockCustodyService) GetAddresses(ctx context.Context) ([]string, error) {
	return []string{"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266", "0x70997970C51812dc3A010C7d01b50e0d17dc79C8"}, nil
}

func (m *MockCustodyService) Transfer(ctx context.Context, intent domain.TransferIntent) (*domain.TransferResult, error) {
	if m.TransferFunc != nil {
		return m.TransferFunc(ctx, intent)
	}
	return &domain.TransferResult{TxHash: "0xMockHash", Status: "broadcasted"}, nil
}

func TestHandleTransferFromQR(t *testing.T) {
	logger := zap.NewNop()

	t.Run("valid transfer request from QR", func(t *testing.T) {
		// Mock service to verify parameters passed to it
		var capturedIntent domain.TransferIntent
		mockService := &MockCustodyService{
			TransferFunc: func(ctx context.Context, intent domain.TransferIntent) (*domain.TransferResult, error) {
				capturedIntent = intent
				return &domain.TransferResult{TxHash: "0xTestTxHash", Status: "broadcasted"}, nil
			},
		}
		server := NewServer(logger, mockService)

		// Create a valid EMVCo QR code payload using the generator helper
		validPayload, err := buildEMVCoQRContent(X9APaymentRequest{
			Address:  "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
			Amount:   "10.50",
			Currency: "USDC",
			Network:  "ethereum",
			Memo:     "invoice-789",
		})
		if err != nil {
			t.Fatalf("failed to build test QR content: %v", err)
		}

		body := `{"payload":"` + validPayload + `","from":"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"}`
		req := httptest.NewRequest("POST", "/transfer/qr", strings.NewReader(body))
		w := httptest.NewRecorder()
		server.handleTransferFromQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusOK {
			t.Errorf("expected status OK, got %v", resp.Status)
		}

		// Verify fields extracted from QR code were transformed correctly
		if capturedIntent.From != "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266" {
			t.Errorf("expected from 0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266, got %s", capturedIntent.From)
		}
		if capturedIntent.To != "0x70997970C51812dc3A010C7d01b50e0d17dc79C8" {
			t.Errorf("expected to 0x70997970C51812dc3A010C7d01b50e0d17dc79C8, got %s", capturedIntent.To)
		}
		// 10.50 * 1000000 = 10500000
		if capturedIntent.Amount != "10500000" {
			t.Errorf("expected amount 10500000, got %s", capturedIntent.Amount)
		}
		if capturedIntent.Currency != "USDC" {
			t.Errorf("expected currency USDC, got %s", capturedIntent.Currency)
		}
	})

	t.Run("invalid from address", func(t *testing.T) {
		server := NewServer(logger, &MockCustodyService{})
		body := `{"payload":"some-payload","from":"invalid-address"}`
		req := httptest.NewRequest("POST", "/transfer/qr", strings.NewReader(body))
		w := httptest.NewRecorder()
		server.handleTransferFromQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %v", resp.Status)
		}
	})

	t.Run("invalid QR payload format", func(t *testing.T) {
		server := NewServer(logger, &MockCustodyService{})
		body := `{"payload":"invalid-payload-without-crc-format","from":"0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"}`
		req := httptest.NewRequest("POST", "/transfer/qr", strings.NewReader(body))
		w := httptest.NewRecorder()
		server.handleTransferFromQR(w, req)

		resp := w.Result()
		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("expected status BadRequest, got %v", resp.Status)
		}
	})
}
