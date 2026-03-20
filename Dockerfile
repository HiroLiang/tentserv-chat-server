# =====================================================
# Build Go binary
# =====================================================
FROM golang:1.26.1-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN go install github.com/swaggo/swag/cmd/swag@latest && \
    swag init -g cmd/server/main.go -o swag-docs

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o goat-server ./cmd/server

# =====================================================
# Run Stage (mount Volume while run it)
# =====================================================
FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/goat-server .

EXPOSE 8080

CMD ["./goat-server"]