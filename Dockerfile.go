FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY . .
RUN go mod download
RUN go build -o bin/api-gateway cmd/api-gateway/main.go

FROM alpine:latest
WORKDIR /app

RUN apk add --no-cache netcat-openbsd

COPY --from=builder /app/bin/api-gateway .
COPY wait-for.sh .
COPY .env .

CMD ["./wait-for.sh", "scoring-service", "50051", "./api-gateway"]