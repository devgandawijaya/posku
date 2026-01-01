# Build stage
FROM golang:1.23-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# build main.go yang ada di folder "cmd/"
RUN CGO_ENABLED=0 GOOS=linux go build -o /posku ./cmd

# Final stage
FROM alpine:latest
RUN apk --no-cache add ca-certificates

WORKDIR /root/
COPY --from=builder /posku .
COPY .env.example .env

EXPOSE 2040
CMD ["./posku"]
