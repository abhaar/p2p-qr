# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

ENV GOWORK=off

# Copy the dependency shared package
COPY shared/ ./shared/

# Copy the protocol service package code
COPY services/blockchain/networks/evm/protocol/ ./services/blockchain/networks/evm/protocol/

# Set working directory to the protocol service module
WORKDIR /app/services/blockchain/networks/evm/protocol

# Download dependencies and build the binary
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/protocol-service cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/protocol-service .

EXPOSE 50065

ENTRYPOINT ["./protocol-service"]
