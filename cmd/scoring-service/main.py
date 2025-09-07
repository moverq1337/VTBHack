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
from collections import defaultdict

# Настройка логирования
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('scoring_service.log')
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

        # Улучшенные паттерны для поиска опыта работы
        experience_patterns = [
            r'Опыт работы.*?(\d+)[^\d]*год',
            r'(\d+)[^\d]*лет.*?опыт',
            r'стаж.*?(\d+)[^\d]*год',
            r'работаю.*?(\d+)[^\d]*год',
            r'experience.*?(\d+)[^\d]*year'
        ]

        max_experience = 0
        for pattern in experience_patterns:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                exp = int(match.group(1))
                if exp > max_experience:
                    max_experience = exp

        # Если не нашли цифры, попробуем определить по датам
        if max_experience == 0:
            date_pattern = r'(\d{4})\s*[-—]\s*(\d{4}|настоящее|н\.в\.|сейчас)'
            dates = re.findall(date_pattern, text)
            if dates:
                current_year = datetime.now().year
                total_experience = 0
                for start_year, end_year in dates:
                    try:
                        start = int(start_year)
                        end = current_year if end_year in ['настоящее', 'н.в.', 'сейчас'] else int(end_year)
                        exp = end - start
                        total_experience += exp
                    except:
                        continue

                if total_experience > 0:
                    max_experience = total_experience / len(dates)  # Средний опыт

        logger.info(f"Найден опыт: {max_experience} лет")
        return max_experience

    def extract_skills(self, text):
        """Извлечение навыков из текста резюме"""
        logger.info("Извлечение навыков из резюме")

        # Расширенный список навыков
        skill_categories = {
            'programming': ['python', 'java', 'javascript', 'c#', 'c++', 'php', 'ruby', 'go', 'rust'],
            'web': ['html', 'css', 'react', 'angular', 'vue', 'django', 'flask', 'node.js', 'express'],
            'database': ['sql', 'mysql', 'postgresql', 'mongodb', 'redis', 'oracle'],
            'devops': ['docker', 'kubernetes', 'jenkins', 'git', 'ci/cd', 'ansible', 'terraform'],
            'os': ['linux', 'windows', 'macos', 'ubuntu', 'debian', 'centos'],
            'networking': ['tcp/ip', 'dns', 'dhcp', 'vpn', 'lan', 'wan'],
            'cloud': ['aws', 'azure', 'google cloud', 'gcp', 'digitalocean'],
            'soft': ['лидерство', 'коммуникация', 'аналитика', 'решение проблем', 'тайм-менеджмент']
        }

        found_skills = defaultdict(list)
        text_lower = text.lower()

        for category, skills in skill_categories.items():
            for skill in skills:
                if re.search(r'\b' + re.escape(skill) + r'\b', text_lower):
                    found_skills[category].append(skill)

        logger.info(f"Найдены навыки: {dict(found_skills)}")
        return dict(found_skills)

    def extract_education(self, text):
        """Извлечение образования из текста резюме"""
        logger.info("Извлечение образования из резюме")

        education_patterns = [
            r'высшее образование',
            r'среднее специальное',
            r'неоконченное высшее',
            r'бакалавр',
            r'магистр',
            r'кандидат наук',
            r'доктор наук'
        ]

        education_levels = []
        for pattern in education_patterns:
            if re.search(pattern, text, re.IGNORECASE):
                education_levels.append(pattern)

        # Извлечение учебных заведений
        universities = []
        uni_patterns = [
            r'([А-Я][а-я]+\s*(университет|институт|академия))',
            r'([А-Я][а-я]+\s*государственный\s*(университет|институт))',
            r'(МГУ|СПбГУ|МФТИ|МГТУ|ВШЭ)'
        ]

        for pattern in uni_patterns:
            matches = re.finditer(pattern, text, re.IGNORECASE)
            for match in matches:
                universities.append(match.group(0))

        result = {
            "levels": education_levels,
            "institutions": list(set(universities))  # Убираем дубликаты
        }

        logger.info(f"Найдено образование: {result}")
        return result

    def ParseResume(self, request, context):
        """Парсинг резюме и извлечение структурированных данных"""
        logger.info(f"Начало парсинга резюме, длина текста: {len(request.text)} символов")

        text = request.text

        try:
            # Извлечение опыта работы
            experience = self.extract_experience(text)

            # Извлечение навыков
            skills = self.extract_skills(text)

            # Извлечение образования
            education = self.extract_education(text)

            # Извлечение языков
            languages = ['Русский']  # По умолчанию
            lang_patterns = {
                'Английский': r'английский',
                'Немецкий': r'немецкий',
                'Французский': r'французский',
                'Испанский': r'испанский',
                'Китайский': r'китайский'
            }

            for lang, pattern in lang_patterns.items():
                if re.search(pattern, text, re.IGNORECASE):
                    languages.append(lang)

            parsed_data = {
                "skills": skills,
                "experience": experience,
                "education": education,
                "languages": languages
            }

            logger.info(f"Результаты парсинга: {parsed_data}")
            return nlp_pb2.ParseResponse(parsed_data=json.dumps(parsed_data, ensure_ascii=False))

        except Exception as e:
            logger.error(f"Ошибка при парсинге резюме: {e}")
            # Возвращаем пустые данные в случае ошибки
            return nlp_pb2.ParseResponse(parsed_data=json.dumps({
                "skills": {},
                "experience": 0,
                "education": {"levels": [], "institutions": []},
                "languages": ["Русский"]
            }, ensure_ascii=False))

    def MatchResumeVacancy(self, request, context):
        """Сопоставление резюме с вакансией с улучшенным анализом"""
        logger.info(f"Сопоставление резюме с вакансией, длина текстов: {len(request.resume_text)}/{len(request.vacancy_text)}")

        resume_text = request.resume_text
        vacancy_text = request.vacancy_text

        try:
            # Базовое сравнение на основе эмбеддингов
            resume_embedding = sentence_model.encode([resume_text])
            vacancy_embedding = sentence_model.encode([vacancy_text])
            base_score = cosine_similarity(resume_embedding, vacancy_embedding)[0][0]

            # Дополнительные факторы для улучшения оценки
            additional_score = 0

            # Проверка соответствия навыков
            resume_skills = self.extract_skills(resume_text)
            vacancy_skills = self.extract_skills(vacancy_text)

            # Считаем совпадение навыков
            matched_skills = 0
            total_skills = 0

            for category, skills in vacancy_skills.items():
                for skill in skills:
                    total_skills += 1
                    if any(s in str(resume_skills.values()).lower() for s in skill.lower().split()):
                        matched_skills += 1

            skill_match_ratio = matched_skills / total_skills if total_skills > 0 else 0

            # Проверка соответствия опыта
            resume_exp = self.extract_experience(resume_text)
            vacancy_exp = self.extract_experience(vacancy_text)

            exp_match = 1 if resume_exp >= vacancy_exp else resume_exp / vacancy_exp

            # Комбинированная оценка
            final_score = 0.5 * base_score + 0.3 * skill_match_ratio + 0.2 * exp_match
            final_score = max(0, min(1, final_score))  # Нормализуем от 0 до 1

            logger.info(f"Базовый score: {base_score:.2f}, Совпадение навыков: {skill_match_ratio:.2f}, Совпадение опыта: {exp_match:.2f}")
            logger.info(f"Итоговый score: {final_score:.2f}")

            return nlp_pb2.MatchResponse(score=final_score)

        except Exception as e:
            logger.error(f"Ошибка при сопоставлении: {e}")
            return nlp_pb2.MatchResponse(score=0.0)

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