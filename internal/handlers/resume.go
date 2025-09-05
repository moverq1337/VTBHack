package handlers

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/ledongthuc/pdf" // Для парсинга PDF
	"github.com/moverq1337/VTBHack/internal/models"
	"github.com/moverq1337/VTBHack/internal/pb"
	"github.com/moverq1337/VTBHack/internal/utils"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

var log = logrus.New()

func SetupResumeRoutes(r *gin.Engine, db *gorm.DB) {
	r.POST("/upload/resume", func(c *gin.Context) { UploadResume(c, db) })
	r.POST("/upload/vacancy", func(c *gin.Context) { UploadVacancy(c, db) })
	r.POST("/analyze/resume", func(c *gin.Context) { AnalyzeResume(c, db) }) // Для анализа и сопоставления
}

// internal/handlers/resume.go
// ... (импорты)

func UploadResume(c *gin.Context, db *gorm.DB) {
	log.Info("Начало загрузки резюме")

	// Создаем папку hr-ai на Яндекс.Диске если ее нет
	if err := utils.CreateFolder("hr-ai"); err != nil {
		log.WithError(err).Warn("Не удалось создать папку hr-ai на Яндекс.Диске")
	}

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
		FileURL:     diskURL, // Сохраняем URL файла на Яндекс.Диске
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

func UploadVacancy(c *gin.Context, db *gorm.DB) {
	var vacancy models.Vacancy
	if err := c.ShouldBindJSON(&vacancy); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Неверный формат данных"})
		return
	}

	vacancy.ID = uuid.New()
	if err := db.Create(&vacancy).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения в БД"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"vacancy_id": vacancy.ID.String()})
}

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
	if err := db.First(&resume, req.ResumeID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Резюме не найдено"})
		return
	}

	var vacancy models.Vacancy
	if err := db.First(&vacancy, req.VacancyID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Вакансия не найдена"})
		return
	}

	conn, err := grpc.Dial(":50051", grpc.WithInsecure())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка gRPC-соединения"})
		return
	}
	defer conn.Close()

	client := pb.NewNLPServiceClient(conn)

	// Парсинг навыков
	parseResp, err := client.ParseResume(context.Background(), &pb.ParseRequest{Text: resume.Text})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка парсинга NLP"})
		return
	}

	// Сопоставление
	matchResp, err := client.MatchResumeVacancy(context.Background(), &pb.MatchRequest{
		ResumeText:  resume.Text,
		VacancyText: vacancy.Requirements + " " + vacancy.Responsibilities,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сопоставления"})
		return
	}

	resume.ParsedData = parseResp.ParsedData
	if err := db.Save(&resume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка обновления БД"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"parsed_data": parseResp.ParsedData,
		"match_score": fmt.Sprintf("%.2f%%", matchResp.Score*100),
	})
}
