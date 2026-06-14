package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
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

		// Verify file was saved locally as HTML
		expectedFile := filepath.Join("qrcodes", "qr_0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266_10.5.html")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("expected QR code HTML file to be saved at %s", expectedFile)
		} else {
			content, err := os.ReadFile(expectedFile)
			if err != nil {
				t.Fatalf("failed to read saved HTML file: %v", err)
			}
			htmlStr := string(content)
			if !strings.Contains(htmlStr, payloadStr) {
				t.Errorf("expected saved HTML file to contain the payload string")
			}
			if !strings.Contains(htmlStr, "data:image/png;base64,") {
				t.Errorf("expected saved HTML file to contain the base64 image data")
			}
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

		// Verify file was saved locally as HTML
		expectedFile := filepath.Join("qrcodes", "qr_0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266_123.45.html")
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("expected QR code HTML file to be saved at %s", expectedFile)
		} else {
			content, err := os.ReadFile(expectedFile)
			if err != nil {
				t.Fatalf("failed to read saved HTML file: %v", err)
			}
			htmlStr := string(content)
			if !strings.Contains(htmlStr, payloadStr) {
				t.Errorf("expected saved HTML file to contain the payload string")
			}
			if !strings.Contains(htmlStr, "data:image/png;base64,") {
				t.Errorf("expected saved HTML file to contain the base64 image data")
			}
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
