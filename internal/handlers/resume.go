package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf"
	"github.com/moverq1337/VTBHack/internal/models"
	"github.com/moverq1337/VTBHack/internal/pb"
	"github.com/moverq1337/VTBHack/internal/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

var log = logrus.New()

// UploadResume обрабатывает загрузку резюме
func UploadResume(c *gin.Context, db *gorm.DB) {
	log.Info("Начало загрузки резюме")

	file, err := c.FormFile("resume")
	if err != nil {
		log.WithError(err).Error("Ошибка загрузки файла")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не загружен: " + err.Error()})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".pdf" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Поддерживается только PDF формат"})
		return
	}

	candidateID := uuid.New()
	filePath := filepath.Join("uploads", candidateID.String()+ext)
	if err := os.MkdirAll("uploads", 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка создания директории"})
		return
	}
	if err := c.SaveUploadedFile(file, filePath); err != nil {
		log.WithError(err).Error("Ошибка сохранения файла")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения файла"})
		return
	}
	defer os.Remove(filePath) // Удаляем временный файл после обработки

	// Извлекаем текст из PDF
	f, r, err := pdf.Open(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка открытия PDF"})
		return
	}
	defer f.Close()

	b, err := r.GetPlainText()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка извлечения текста из PDF"})
		return
	}

	buf := &bytes.Buffer{}
	buf.ReadFrom(b)
	text := buf.String()

	// Загружаем на Яндекс.Диск
	diskURL, err := utils.UploadToYandexDisk(filePath, file.Filename)
	if err != nil {
		log.WithError(err).Error("Ошибка загрузки на Яндекс.Диск")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка загрузки на Яндекс.Диск: " + err.Error()})
		return
	}

	resume := models.Resume{
		ID:          uuid.New(),
		CandidateID: candidateID,
		Text:        text,
		ParsedData:  "{}",
		FileURL:     diskURL,
	}
	if err := db.Create(&resume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения в БД"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"candidate_id": candidateID.String(),
		"file_url":     diskURL,
		"resume_id":    resume.ID.String(),
	})
}

// UploadVacancy обрабатывает загрузку вакансии
func UploadVacancy(c *gin.Context, db *gorm.DB) {
	type VacancyRequest struct {
		Title            string `json:"title"`
		Requirements     string `json:"requirements"`
		Responsibilities string `json:"responsibilities"`
		Region           string `json:"region"`
		City             string `json:"city"`
		EmploymentType   string `json:"employment_type"`
		WorkSchedule     string `json:"work_schedule"`
		Experience       string `json:"experience"`
		Education        string `json:"education"`
		SalaryMin        int    `json:"salary_min"`
		SalaryMax        int    `json:"salary_max"`
		Languages        string `json:"languages"`
		Skills           string `json:"skills"`
	}

	var req VacancyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	vacancy := models.Vacancy{
		ID:               uuid.New(),
		Title:            req.Title,
		Requirements:     req.Requirements,
		Responsibilities: req.Responsibilities,
		Region:           req.Region,
		City:             req.City,
		EmploymentType:   req.EmploymentType,
		WorkSchedule:     req.WorkSchedule,
		Experience:       req.Experience,
		Education:        req.Education,
		SalaryMin:        req.SalaryMin,
		SalaryMax:        req.SalaryMax,
		Languages:        req.Languages,
		Skills:           req.Skills,
		CreatedAt:        time.Now(),
	}

	if err := db.Create(&vacancy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения в БД"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"vacancy_id": vacancy.ID.String(),
		"title":      vacancy.Title,
	})
}

// AnalyzeResume обрабатывает анализ резюме
func AnalyzeResume(c *gin.Context, db *gorm.DB) {
	type AnalyzeRequest struct {
		ResumeID  uuid.UUID `json:"resume_id"`
		VacancyID uuid.UUID `json:"vacancy_id"`
	}

	var req AnalyzeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат запроса"})
		return
	}

	var resume models.Resume
	if err := db.First(&resume, "id = ?", req.ResumeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Резюме не найдено"})
		return
	}

	var vacancy models.Vacancy
	if err := db.First(&vacancy, "id = ?", req.VacancyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Вакансия не найдена"})
		return
	}

	// Подключаемся к NLP-сервису
	grpcHost := os.Getenv("GRPC_HOST")
	if grpcHost == "" {
		grpcHost = "scoring-service"
	}
	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	conn, err := grpc.Dial(fmt.Sprintf("%s:%s", grpcHost, grpcPort), grpc.WithInsecure())
	if err != nil {
		log.WithError(err).Error("Ошибка gRPC-соединения")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подключения к сервису анализа"})
		return
	}
	defer conn.Close()

	client := pb.NewNLPServiceClient(conn)

	// Сопоставление с вакансией (без предварительного парсинга)
	matchResp, err := client.MatchResumeVacancy(context.Background(), &pb.MatchRequest{
		ResumeText:  resume.Text,
		VacancyText: fmt.Sprintf("%s %s %s %s", vacancy.Title, vacancy.Requirements, vacancy.Responsibilities, vacancy.Skills),
	})
	if err != nil {
		log.WithError(err).Error("Ошибка сопоставления")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сопоставления с вакансией"})
		return
	}

	// Сохраняем результаты анализа
	analysisResult := models.AnalysisResult{
		ID:         uuid.New(),
		ResumeID:   resume.ID,
		VacancyID:  vacancy.ID,
		MatchScore: float64(matchResp.Score),
		CreatedAt:  time.Now(),
	}

	if err := db.Create(&analysisResult).Error; err != nil {
		log.WithError(err).Error("Ошибка сохранения результатов анализа")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результатов"})
		return
	}

	// Формируем ответ
	response := gin.H{
		"analysis_id":  analysisResult.ID.String(),
		"resume_id":    resume.ID.String(),
		"vacancy_id":   vacancy.ID.String(),
		"match_score":  fmt.Sprintf("%.2f%%", matchResp.Score*100),
		"candidate_id": resume.CandidateID.String(),
		"created_at":   analysisResult.CreatedAt,
	}

	c.JSON(http.StatusOK, response)
}
