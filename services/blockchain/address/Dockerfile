# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

ENV GOWORK=off

# Copy the dependency shared package
COPY shared/ ./shared/

# Copy the address service package code
COPY services/blockchain/address/ ./services/blockchain/address/

# Set working directory to the address service module
WORKDIR /app/services/blockchain/address

# Download dependencies and build the binary
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/address-service cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/address-service .

EXPOSE 50064

ENTRYPOINT ["./address-service"]
