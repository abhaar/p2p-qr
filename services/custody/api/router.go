package api

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/p2p/custody/v2/internal/domain"
)

// Server holds dependencies for the custody API handlers.
type Server struct {
	logger  *zap.Logger
	custody domain.CustodyService
}

// NewServer creates a new Server.
func NewServer(logger *zap.Logger, custody domain.CustodyService) *Server {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}
	return &Server{
		logger:  logger,
		custody: custody,
	}
}

// RegisterRoutes registers the custody API routes on the given mux.
// It uses the Go 1.22+ method-aware pattern syntax.
func (s *Server) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /balance/{address}", s.handleGetBalance)
	mux.HandleFunc("POST /transfer", s.handleTransfer)
	mux.HandleFunc("GET /payment-request/qr", s.handleGeneratePaymentQR)
	mux.HandleFunc("POST /payment-request/qr", s.handleGeneratePaymentQR)
	mux.HandleFunc("POST /transfer/qr", s.handleTransferFromQR)
}
