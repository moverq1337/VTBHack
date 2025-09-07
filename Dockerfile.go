FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o bin/api-gateway cmd/api-gateway/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/bin/api-gateway .
COPY .env .
COPY frontend ./frontend 
CMD ["./api-gateway"]