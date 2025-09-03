## Архитектура
- API Gateway (Gin): Обработка HTTP-запросов.
- Resume Service: Парсинг резюме (Go + Python via gRPC).
- Interview Service: Голосовые интервью (Go + SpeechKit).
- Scoring Service: Оценка (Python NLP).
- Report Service: Отчеты (Go).
- БД: PostgreSQL.
- Кэш: Redis.
- Очереди: Kafka.

## .env 
- DB_URL=postgres://postgres:password@postgres:5432/hrdb?sslmode=disable
- GRPC_PORT=:50051
- HTTP_PORT=:8080
- REDIS_ADDR=redis:6379
- KAFKA_BROKERS=kafka:9092