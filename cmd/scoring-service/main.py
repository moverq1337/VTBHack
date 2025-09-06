# scoring-service/main.py
import grpc
from concurrent import futures
import time
import nlp_pb2
import nlp_pb2_grpc
import json

class NLPService(nlp_pb2_grpc.NLPServiceServicer):
    def ParseResume(self, request, context):
        """Парсинг резюме и извлечение структурированных данных"""
        # Здесь реализуйте парсинг резюме
        parsed_data = {
            "skills": ["Go", "Python", "PostgreSQL"],
            "experience": 5,
            "education": "Высшее",
            "languages": ["Русский", "Английский"]
        }

        return nlp_pb2.ParseResponse(parsed_data=json.dumps(parsed_data))

    def MatchResumeVacancy(self, request, context):
        """Сопоставление резюме с вакансией"""
        # Простая реализация сопоставления
        # В реальности здесь должен быть сложный алгоритм

        resume_text = request.resume_text.lower()
        vacancy_text = request.vacancy_text.lower()

        # Простой пример: считаем совпадение ключевых слов
        keywords = ["go", "python", "postgresql", "docker", "kubernetes"]
        matches = 0

        for keyword in keywords:
            if keyword in resume_text and keyword in vacancy_text:
                matches += 1

        score = min(matches / len(keywords), 1.0)  # Оценка от 0 до 1

        return nlp_pb2.MatchResponse(score=score)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    nlp_pb2_grpc.add_NLPServiceServicer_to_server(NLPService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    print("gRPC server started on port 50051")
    server.wait_for_termination()

if __name__ == '__main__':
    serve()