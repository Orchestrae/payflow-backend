# Dockerfile.dev

FROM golang:1.22-alpine

# Install Air for hot-reloading
RUN go install github.com/cosmtrek/air@latest

WORKDIR /app
COPY . .

# Ensure dependencies are available
RUN go mod tidy

EXPOSE 8080
CMD ["air"]