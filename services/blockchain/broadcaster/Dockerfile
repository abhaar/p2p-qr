# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

ENV GOWORK=off

# Copy the dependency shared package
COPY shared/ ./shared/

# Copy the broadcaster service package code
COPY services/blockchain/broadcaster/ ./services/blockchain/broadcaster/

# Copy the signer service package code
COPY services/blockchain/signer/ ./services/blockchain/signer/

# Set working directory to the broadcaster service module
WORKDIR /app/services/blockchain/broadcaster

# Download dependencies and build the binary
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/broadcaster-service cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/broadcaster-service .

EXPOSE 50067

ENTRYPOINT ["./broadcaster-service"]
