# Build stage
FROM golang:1.25.5-alpine AS builder

WORKDIR /app

ENV GOWORK=off

# Copy the dependency shared package
COPY shared/ ./shared/

# Copy the custody service package code
COPY services/custody/ ./services/custody/

# Set working directory to the custody service module
WORKDIR /app/services/custody

# Download dependencies and build the binary
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -o /app/custody-service cmd/main.go

# Runtime stage
FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

# Copy the binary from the build stage
COPY --from=builder /app/custody-service .

EXPOSE 8080

ENTRYPOINT ["./custody-service"]
