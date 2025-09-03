FROM python:3.11-slim

WORKDIR /app

COPY cmd/scoring-service/requirements.txt .
RUN pip3 install --no-cache-dir -r requirements.txt

RUN pip3 install spacy

RUN python3 -m spacy download ru_core_news_lg

COPY cmd/scoring-service/ .

CMD ["python3", "main.py"]
