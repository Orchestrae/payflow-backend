# syntax=docker/dockerfile:1
# Production Dockerfile - builds binary at image build time (works on Railway)

# Build stage
FROM golang:1.26-alpine AS builder

ENV GOTOOLCHAIN=auto

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /payflow ./cmd/server

# Run stage - minimal image
FROM alpine:3.19

RUN apk add --no-cache ca-certificates

WORKDIR /app

COPY --from=builder /payflow .

EXPOSE 8080

CMD ["./payflow"]
