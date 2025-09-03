import json
import grpc
from concurrent import futures
import spacy
from docx import Document
from sentence_transformers import SentenceTransformer, util
import nlp_pb2
import nlp_pb2_grpc

nlp = spacy.load("ru_core_news_lg")
model = SentenceTransformer('paraphrase-multilingual-MiniLM-L12-v2')

class NLPService(nlp_pb2_grpc.NLPServiceServicer):
    def ParseResume(self, request, context):
        text = request.text
        # Парсинг DOCX, если передан путь к файлу
        if text.endswith('.docx'):
            doc = Document(text)
            text = '\n'.join([para.text for para in doc.paragraphs])

        # Извлечение навыков и опыта с помощью spaCy
        doc = nlp(text)
        skills = [ent.text for ent in doc.ents if ent.label_ == "SKILL"]  # Предполагаем кастомные метки или фильтрацию
        experience = [ent.text for ent in doc.ents if ent.label_ == "DATE"]

        # Формирование JSON-ответа
        parsed_data = {
            "skills": skills,
            "experience": experience,
            "raw_text": text
        }
        return nlp_pb2.ParseResponse(parsed_data=json.dumps(parsed_data))

    def MatchResumeVacancy(self, request, context):
        resume_text = request.resume_text
        vacancy_text = request.vacancy_text
        # Вычисление эмбеддингов
        resume_embedding = model.encode(resume_text, convert_to_tensor=True)
        vacancy_embedding = model.encode(vacancy_text, convert_to_tensor=True)
        # Косинусное сходство
        score = util.cos_sim(resume_embedding, vacancy_embedding).item()
        return nlp_pb2.MatchResponse(score=score)

def serve():
    server = grpc.server(futures.ThreadPoolExecutor(max_workers=10))
    nlp_pb2_grpc.add_NLPServiceServicer_to_server(NLPService(), server)
    server.add_insecure_port('[::]:50051')
    server.start()
    server.wait_for_termination()

if __name__ == '__main__':
    serve()