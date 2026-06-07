# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

ENV GOWORK=off

# Copy the dependency shared package
COPY shared/ ./shared/

# Copy the indexer service package code
COPY services/blockchain/indexer/ ./services/blockchain/indexer/

# Set working directory to the indexer service module
WORKDIR /app/services/blockchain/indexer

# Download dependencies and build the binary
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/indexer-service cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/indexer-service .

EXPOSE 50066

ENTRYPOINT ["./indexer-service"]
