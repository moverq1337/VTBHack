FROM python:3.11-slim

WORKDIR /app
COPY cmd/scoring-service/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt
COPY cmd/scoring-service/ .
CMD ["python", "main.py"]  # Python-сервер gRPC