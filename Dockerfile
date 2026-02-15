# syntax=docker/dockerfile:1
FROM golang:1.24-alpine

WORKDIR /app

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .

RUN go install github.com/air-verse/air@v1.52.3

EXPOSE 8080

CMD ["air", "-c", "payflow.air.toml"]