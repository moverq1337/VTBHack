# scoring-service/main.py
import grpc
from concurrent import futures
import time
import nlp_pb2
import nlp_pb2_grpc
import json
import spacy
import re
import logging
from datetime import datetime
from sentence_transformers import SentenceTransformer
from sklearn.metrics.pairwise import cosine_similarity
import numpy as np

# Настройка логирования
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),  # Вывод в консоль
        logging.FileHandler('scoring_service.log')  # Запись в файл
    ]
)
logger = logging.getLogger(__name__)

# Загрузка моделей
logger.info("Загрузка моделей NLP...")
try:
    nlp = spacy.load("ru_core_news_sm")
    sentence_model = SentenceTransformer('paraphrase-multilingual-MiniLM-L12-v2')
    logger.info("Модели успешно загружены")
except Exception as e:
    logger.error(f"Ошибка загрузки моделей: {e}")
    raise

class NLPService(nlp_pb2_grpc.NLPServiceServicer):
    def extract_experience(self, text):
        """Извлечение опыта работы из текста резюме"""
        logger.info("Извлечение опыта работы из резюме")
        experience_patterns = [
            r'(\d+)\s*год[а]?',
            r'(\d+)\s*лет',
            r'Опыт работы.*?(\d+)'
        ]

        for pattern in experience_patterns:
            match = re.search(pattern, text, re.IGNORECASE)
            if match:
                exp = int(match.group(1))
                logger.info(f"Найден опыт: {exp} лет")
                return exp
        logger.warning("Опыт работы не найден")
        return 0

    def extract_skills(self, text):
        """Извлечение навыков из текста резюме"""
        logger.info("Извлечение навыков из резюме")
        skill_keywords = [
            'javascript', 'python', 'java', 'html', 'css', 'react',
            'node.js', 'sql', 'nosql', 'docker', 'kubernetes', 'aws',
            'azure', 'git', 'linux', 'windows', 'администрирование',
            'поддержка', 'настройка', 'разработка', 'проектирование'
        ]

        found_skills = []
        for skill in skill_keywords:
            if re.search(r'\b' + re.escape(skill) + r'\b', text, re.IGNORECASE):
                found_skills.append(skill)

        logger.info(f"Найдены навыки: {found_skills}")
        return found_skills

    def extract_education(self, text):
        """Извлечение образования из текста резюме"""
        logger.info("Извлечение образования из резюме")
        education_keywords = [
            'высшее', 'неоконченное высшее', 'среднее специальное',
            'бакалавр', 'магистр', 'кандидат наук', 'доктор наук'
        ]

        education_levels = []
        for edu in education_keywords:
            if re.search(r'\b' + re.escape(edu) + r'\b', text, re.IGNORECASE):
                education_levels.append(edu)

        logger.info(f"Найдено образование: {education_levels}")
        return education_levels

    def ParseResume(self, request, context):
        """Парсинг резюме и извлечение структурированных данных"""
        logger.info(f"Начало парсинга резюме, длина текста: {len(request.text)} символов")

        text = request.text

        # Извлечение опыта работы
        experience = self.extract_experience(text)

        # Извлечение навыков
        skills = self.extract_skills(text)

        # Извлечение образования
        education = self.extract_education(text)

        # Извлечение языков
        languages = ['Русский']  # По умолчанию
        if re.search(r'английский', text, re.IGNORECASE):
            languages.append('Английский')

        parsed_data = {
            "skills": skills,
            "experience": experience,
            "education": education[0] if education else "Не указано",
            "languages": languages
        }

        logger.info(f"Результаты парсинга: {parsed_data}")
        return nlp_pb2.ParseResponse(parsed_data=json.dumps(parsed_data))

    def MatchResumeVacancy(self, request, context):
        """Сопоставление резюме с вакансией"""
        logger.info(f"Сопоставление резюме с вакансией, длина текстов: {len(request.resume_text)}/{len(request.vacancy_text)}")

        resume_text = request.resume_text
        vacancy_text = request.vacancy_text

        # Логирование начала текстов для отладки
        logger.info(f"Начало резюме: {resume_text[:200]}...")
        logger.info(f"Начало вакансии: {vacancy_text[:200]}...")

        # Получаем эмбеддинги для резюме и вакансии
        logger.info("Создание эмбеддингов...")
        resume_embedding = sentence_model.encode([resume_text])
        vacancy_embedding = sentence_model.encode([vacancy_text])

        # Вычисляем косинусное сходство
        logger.info("Вычисление косинусного сходства...")
        score = cosine_similarity(resume_embedding, vacancy_embedding)[0][0]

        # Нормализуем оценку от 0 до 1
        normalized_score = max(0, min(1, score))

        logger.info(f"Результат сопоставления: {normalized_score:.2f}")
        return nlp_pb2.MatchResponse(score=normalized_score)

def serve():
    logger.info("Запуск gRPC сервера на порту 50051")
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    nlp_pb2_grpc.add_NLPServiceServicer_to_server(NLPService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    logger.info("gRPC сервер успешно запущен")

    try:
        server.wait_for_termination()
    except KeyboardInterrupt:
        logger.info("Остановка сервера...")
        server.stop(0)

if __name__ == '__main__':
    serve()