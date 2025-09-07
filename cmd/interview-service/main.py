import asyncio
import websockets
import json
import base64
import logging
from typing import Dict
import uuid
import io
import wave
import tempfile
import os
import aiohttp
from datetime import datetime
import subprocess

# Настройка логирования
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.StreamHandler(),
        logging.FileHandler('interview_service.log')
    ]
)
logger = logging.getLogger(__name__)

class InterviewManager:
    def __init__(self):
        self.sessions: Dict[str, dict] = {}
        self.questions = [
            "Расскажите о вашем опыте работы с Docker и контейнеризацией",
            "Как вы организуете процесс CI/CD в своих проектах?",
            "Какие методы мониторинга и логирования вы используете?",
            "Расскажите о вашем опыте работы с облачными платформами"
        ]
        self.api_key = os.getenv('YANDEX_API_KEY')
        if not self.api_key:
            logger.error("YANDEX_API_KEY not set. Speech services will not work properly.")

    def create_session(self, candidate_id: str, vacancy_id: str) -> str:
        session_id = str(uuid.uuid4())
        self.sessions[session_id] = {
            'candidate_id': candidate_id,
            'vacancy_id': vacancy_id,
            'current_question': 0,
            'answers': [],
            'score': 0,
            'start_time': datetime.now().isoformat()
        }
        logger.info(f"Создана сессия интервью: {session_id}")
        return session_id

    def get_next_question(self, session_id: str) -> str:
        session = self.sessions.get(session_id)
        if not session:
            logger.warning(f"Сессия не найдена: {session_id}")
            return None

        if session['current_question'] < len(self.questions):
            question = self.questions[session['current_question']]
            logger.info(f"Вопрос {session['current_question'] + 1}: {question}")
            return question
        logger.info("Все вопросы заданы")
        return None

    def save_answer(self, session_id: str, answer: str, score: float):
        session = self.sessions.get(session_id)
        if not session:
            logger.error(f"Сессия не найдена при сохранении ответа: {session_id}")
            return False

        question = self.questions[session['current_question']]
        session['answers'].append({
            'question': question,
            'answer': answer,
            'score': score,
            'timestamp': datetime.now().isoformat()
        })
        session['score'] += score
        session['current_question'] += 1

        logger.info(f"Сохранен ответ для сессии {session_id}: score={score}, answer_length={len(answer)}")
        return True

    def get_results(self, session_id: str) -> dict:
        session = self.sessions.get(session_id)
        if not session:
            logger.warning(f"Сессия не найдена при получении результатов: {session_id}")
            return None

        session['end_time'] = datetime.now().isoformat()
        session['duration'] = (datetime.fromisoformat(session['end_time']) -
                              datetime.fromisoformat(session['start_time'])).total_seconds()

        logger.info(f"Интервью завершено: {session_id}, длительность: {session['duration']} сек")
        return session

    async def convert_audio_to_pcm(self, audio_data: bytes) -> bytes:
        """Конвертирует аудио в PCM формат для Yandex SpeechKit"""
        try:
            # Создаем временные файлы для конвертации
            with tempfile.NamedTemporaryFile(suffix='.webm', delete=False) as input_file:
                input_file.write(audio_data)
                input_file.flush()

                output_file_path = input_file.name + '.pcm'

                # Конвертируем в PCM с помощью ffmpeg
                command = [
                    'ffmpeg', '-i', input_file.name,
                    '-f', 's16le', '-acodec', 'pcm_s16le', '-ar', '48000', '-ac', '1',
                    output_file_path, '-y'
                ]

                process = await asyncio.create_subprocess_exec(
                    *command,
                    stdout=asyncio.subprocess.PIPE,
                    stderr=asyncio.subprocess.PIPE
                )

                stdout, stderr = await process.communicate()

                if process.returncode != 0:
                    logger.error(f"Ошибка конвертации аудио: {stderr.decode()}")
                    return None

                # Читаем сконвертированное аудио
                with open(output_file_path, 'rb') as f:
                    pcm_data = f.read()

                # Удаляем временные файлы
                os.unlink(input_file.name)
                os.unlink(output_file_path)

                return pcm_data

        except Exception as e:
            logger.error(f"Ошибка при конвертации аудио: {e}")
            return None

    async def yandex_text_to_speech(self, text: str) -> bytes:
        """Преобразование текста в речь с использованием Yandex SpeechKit TTS"""
        if not self.api_key:
            logger.error("YANDEX_API_KEY not set")
            return None

        url = 'https://tts.api.cloud.yandex.net/speech/v1/tts:synthesize'
        headers = {
            'Authorization': f'Api-Key {self.api_key}'
        }

        data = {
            'text': text,
            'voice': 'alena',
            'emotion': 'neutral',
            'format': 'lpcm',
            'sampleRateHertz': '48000',
        }

        try:
            async with aiohttp.ClientSession() as session:
                async with session.post(url, headers=headers, data=data, timeout=aiohttp.ClientTimeout(total=30)) as response:
                    if response.status == 200:
                        audio_data = await response.read()
                        logger.info(f"Yandex TTS успешен, размер аудио: {len(audio_data)} байт")
                        return audio_data
                    else:
                        error_text = await response.text()
                        logger.error(f"Ошибка Yandex TTS: {response.status}, {error_text}")
                        return None
        except Exception as e:
            logger.error(f"Исключение в Yandex TTS: {e}")
            return None

    async def yandex_speech_to_text(self, audio_data: bytes) -> str:
        """Преобразование речи в текст с использованием Yandex SpeechKit STT"""
        if not self.api_key:
            logger.error("YANDEX_API_KEY not set")
            return "Ошибка: не настроен API ключ", 0.0

        # Конвертируем аудио в PCM формат
        pcm_data = await self.convert_audio_to_pcm(audio_data)
        if not pcm_data:
            return "Ошибка конвертации аудио", 0.0

        url = 'https://stt.api.cloud.yandex.net/speech/v1/stt:recognize'
        headers = {
            'Authorization': f'Api-Key {self.api_key}',
            'Content-Type': 'audio/x-pcm'
        }

        try:
            async with aiohttp.ClientSession() as session:
                async with session.post(url, headers=headers, data=pcm_data, timeout=aiohttp.ClientTimeout(total=30)) as response:
                    if response.status == 200:
                        result = await response.json()
                        transcribed_text = result.get('result', '')

                        # Простая оценка на основе длины текста
                        score = min(len(transcribed_text) / 100, 1.0)

                        logger.info(f"Yandex STT успешен, распознанный текст: '{transcribed_text}', оценка: {score}")
                        return transcribed_text, score
                    else:
                        error_text = await response.text()
                        logger.error(f"Ошибка Yandex STT: {response.status}, {error_text}")
                        return "Ошибка распознавания речи", 0.0
        except Exception as e:
            logger.error(f"Исключение в Yandex STT: {e}")
            return "Ошибка распознавания речи", 0.0

interview_manager = InterviewManager()

async def handle_interview(websocket):
    client_ip = websocket.remote_address[0]
    logger.info(f"Новое подключение от {client_ip}")

    try:
        async for message in websocket:
            data = json.loads(message)
            logger.info(f"Получено сообщение типа: {data['type']}")

            if data['type'] == 'start_interview':
                # Начинаем новое интервью
                session_id = interview_manager.create_session(
                    data['candidate_id'],
                    data['vacancy_id']
                )
                first_question = interview_manager.get_next_question(session_id)

                # Генерируем аудио вопроса с помощью Yandex TTS
                question_audio = await interview_manager.yandex_text_to_speech(first_question)
                if question_audio is None:
                    # Fallback: отправляем текст вопроса без аудио
                    await websocket.send(json.dumps({
                        'type': 'question',
                        'session_id': session_id,
                        'question': first_question,
                        'question_audio': None,
                        'question_number': 1,
                        'total_questions': len(interview_manager.questions)
                    }))
                else:
                    audio_base64 = base64.b64encode(question_audio).decode('utf-8')
                    await websocket.send(json.dumps({
                        'type': 'question',
                        'session_id': session_id,
                        'question': first_question,
                        'question_audio': audio_base64,
                        'question_number': 1,
                        'total_questions': len(interview_manager.questions)
                    }))
                logger.info(f"Отправлен вопрос 1 для сессии {session_id}")

            elif data['type'] == 'audio_response':
                # Обрабатываем аудио ответ
                session_id = data['session_id']
                logger.info(f"Обработка аудио ответа для сессии {session_id}")

                try:
                    audio_data = base64.b64decode(data['audio'])
                    logger.info(f"Получено аудио: {len(audio_data)} байт")

                    # Распознаем речь с помощью Yandex STT
                    transcribed_text, score = await interview_manager.yandex_speech_to_text(audio_data)

                    # Сохраняем ответ
                    interview_manager.save_answer(session_id, transcribed_text, score)

                    # Получаем следующий вопрос
                    next_question = interview_manager.get_next_question(session_id)

                    if next_question:
                        # Генерируем аудио следующего вопроса с помощью Yandex TTS
                        question_audio = await interview_manager.yandex_text_to_speech(next_question)
                        if question_audio is None:
                            # Fallback: отправляем текст вопроса без аудио
                            await websocket.send(json.dumps({
                                'type': 'question',
                                'session_id': session_id,
                                'question': next_question,
                                'question_audio': None,
                                'question_number': interview_manager.sessions[session_id]['current_question'] + 1,
                                'total_questions': len(interview_manager.questions)
                            }))
                        else:
                            audio_base64 = base64.b64encode(question_audio).decode('utf-8')
                            await websocket.send(json.dumps({
                                'type': 'question',
                                'session_id': session_id,
                                'question': next_question,
                                'question_audio': audio_base64,
                                'question_number': interview_manager.sessions[session_id]['current_question'] + 1,
                                'total_questions': len(interview_manager.questions)
                            }))
                        logger.info(f"Отправлен вопрос {interview_manager.sessions[session_id]['current_question'] + 1}")
                    else:
                        # Интервью завершено
                        results = interview_manager.get_results(session_id)
                        await websocket.send(json.dumps({
                            'type': 'interview_completed',
                            'session_id': session_id,
                            'score': results['score'],
                            'answers': results['answers'],
                            'total_score': results['score'] * 25,  # Конвертируем в 100-балльную систему
                            'duration': results['duration']
                        }))
                        logger.info(f"Интервью завершено для сессии {session_id}")

                except Exception as e:
                    logger.error(f"Ошибка обработки аудио: {e}")
                    await websocket.send(json.dumps({
                        'type': 'error',
                        'message': 'Ошибка обработки аудио ответа'
                    }))

    except websockets.exceptions.ConnectionClosed as e:
        logger.info(f"Соединение закрыто: {e}")
    except Exception as e:
        logger.error(f"Ошибка в обработчике интервью: {e}")
        try:
            await websocket.send(json.dumps({
                'type': 'error',
                'message': 'Внутренняя ошибка сервера'
            }))
        except:
            pass

async def health_check():
    """Периодическая проверка здоровья сервиса"""
    while True:
        logger.info(f"Активных сессий: {len(interview_manager.sessions)}")
        await asyncio.sleep(60)

async def main():
    # Запускаем фоновую задачу для проверки здоровья
    asyncio.create_task(health_check())

    # Запускаем WebSocket сервер
    server = await websockets.serve(handle_interview, "0.0.0.0", 8765)
    logger.info("WebSocket сервер запущен на порту 8765")

    try:
        await server.wait_closed()
    except KeyboardInterrupt:
        logger.info("Остановка сервера...")
    except Exception as e:
        logger.error(f"Ошибка сервера: {e}")

if __name__ == "__main__":
    asyncio.run(main())