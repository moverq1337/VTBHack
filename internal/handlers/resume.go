package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/unidoc/unioffice/common/license"
	"google.golang.org/grpc/credentials/insecure"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/moverq1337/VTBHack/internal/models"
	"github.com/moverq1337/VTBHack/internal/pb"
	"github.com/moverq1337/VTBHack/internal/utils"
	"github.com/sirupsen/logrus"
	"github.com/unidoc/unioffice/document"
	"google.golang.org/grpc"
	"gorm.io/gorm"
)

var log = logrus.New()

// UploadResume обрабатывает загрузку резюме в формате DOCX
func UploadResume(c *gin.Context, db *gorm.DB) {
	log.Info("Начало загрузки резюме DOCX")

	file, err := c.FormFile("resume")
	if err != nil {
		log.WithError(err).Error("Ошибка загрузки файла")
		c.JSON(http.StatusBadRequest, gin.H{"error": "Файл не загружен: " + err.Error()})
		return
	}

	ext := filepath.Ext(file.Filename)
	if ext != ".docx" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Поддерживается только DOCX формат"})
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

	// Извлекаем текст из DOCX
	text, err := extractTextFromDOCX(filePath)
	if err != nil {
		log.WithError(err).Error("Ошибка извлечения текста из DOCX")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка извлечения текста из DOCX: " + err.Error()})
		return
	}

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

	// Вызов парсинга резюме
	grpchost := "scoring-service:50051"
	conn, err := grpc.NewClient(grpchost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithError(err).Error("Ошибка gRPC-соединения для парсинга")
	} else {
		defer conn.Close()

		client := pb.NewNLPServiceClient(conn)
		parseResp, err := client.ParseResume(context.Background(), &pb.ParseRequest{
			Text: text,
		})

		if err != nil {
			log.WithError(err).Error("Ошибка парсинга резюме")
		} else {
			// Сохраняем результаты парсинга
			resume.ParsedData = parseResp.ParsedData
		}
	}

	if err := db.Create(&resume).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения в БД"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"candidate_id": candidateID.String(),
		"file_url":     diskURL,
		"resume_id":    resume.ID.String(),
		"text_preview": truncateText(text, 200), // Первые 200 символов для предпросмотра
	})
}

// extractTextFromDOCX извлекает текст из DOCX файла
func extractTextFromDOCX(filePath string) (string, error) {
	apiKey := os.Getenv("UNIDOC_LICENSE_API_KEY")
	if apiKey == "" {
		log.Fatal("UNIDOC_LICENSE_API_KEY environment variable not set")
	}

	// Установка API ключа для UniDoc
	err := license.SetMeteredKey(apiKey)
	if err != nil {
		log.Fatalf("Ошибка инициализации UniDoc license: %v", err)
	}
	doc, err := document.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("ошибка открытия DOCX файла: %v", err)
	}

	var text string
	for _, para := range doc.Paragraphs() {
		for _, run := range para.Runs() {
			text += run.Text()
		}
		text += "\n"
	}

	// Также извлекаем текст из таблиц
	for _, tbl := range doc.Tables() {
		for _, row := range tbl.Rows() {
			for _, cell := range row.Cells() {
				for _, para := range cell.Paragraphs() {
					for _, run := range para.Runs() {
						text += run.Text() + " "
					}
				}
			}
			text += "\n"
		}
	}

	return text, nil
}

// truncateText обрезает текст до указанной длины
func truncateText(text string, maxLength int) string {
	if len(text) <= maxLength {
		return text
	}
	return text[:maxLength] + "..."
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

	grpchost := "scoring-service:50051"

	conn, err := grpc.NewClient(grpchost, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.WithError(err).Error("Ошибка gRPC-соединения")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка подключения к сервису анализа"})
		return
	}
	defer conn.Close()

	client := pb.NewNLPServiceClient(conn)

	// Сопоставление с вакансией
	matchResp, err := client.MatchResumeVacancy(context.Background(), &pb.MatchRequest{
		ResumeText:  resume.Text,
		VacancyText: fmt.Sprintf("%s %s %s %s", vacancy.Title, vacancy.Requirements, vacancy.Responsibilities, vacancy.Skills),
	})
	if err != nil {
		log.WithError(err).Error("Ошибка сопоставления")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сопоставления с вакансией"})
		return
	}

	// Парсим резюме для получения деталей
	parseResp, err := client.ParseResume(context.Background(), &pb.ParseRequest{
		Text: resume.Text,
	})
	if err != nil {
		log.WithError(err).Error("Ошибка парсинга резюме")
	}

	var parsedData map[string]interface{}
	if err := json.Unmarshal([]byte(parseResp.ParsedData), &parsedData); err != nil {
		log.WithError(err).Error("Ошибка разбора JSON данных парсинга")
		parsedData = make(map[string]interface{})
	}

	// Сохраняем результаты анализа
	analysisResult := models.AnalysisResult{
		ID:         uuid.New(),
		ResumeID:   resume.ID,
		VacancyID:  vacancy.ID,
		MatchScore: float64(matchResp.Score),
		Details:    parseResp.ParsedData, // Сохраняем полные данные парсинга
		CreatedAt:  time.Now(),
	}

	if err := db.Create(&analysisResult).Error; err != nil {
		log.WithError(err).Error("Ошибка сохранения результатов анализа")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Ошибка сохранения результатов"})
		return
	}

	// Сохраняем детали анализа
	if skills, ok := parsedData["skills"].(map[string]interface{}); ok {
		for category, skillList := range skills {
			if skillsArr, ok := skillList.([]interface{}); ok {
				for _, skill := range skillsArr {
					analysisDetail := models.AnalysisDetail{
						ID:               uuid.New(),
						AnalysisResultID: analysisResult.ID,
						Category:         "skills",
						Criteria:         category,
						ResumeValue:      fmt.Sprintf("%v", skill),
						VacancyValue:     "",  // Можно добавить проверку наличия в вакансии
						MatchScore:       0.8, // Заглушка
						Weight:           0.3, // Заглушка
						CreatedAt:        time.Now(),
					}
					if err := db.Create(&analysisDetail).Error; err != nil {
						log.WithError(err).Error("Ошибка сохранения деталей анализа")
					}
				}
			}
		}
	}

	// Формируем ответ
	c.JSON(http.StatusOK, gin.H{
		"analysis_id":  analysisResult.ID.String(),
		"resume_id":    resume.ID.String(),
		"vacancy_id":   vacancy.ID.String(),
		"match_score":  fmt.Sprintf("%.2f%%", matchResp.Score*100),
		"candidate_id": resume.CandidateID.String(),
		"created_at":   analysisResult.CreatedAt,
		"details":      parsedData, // Добавляем детали в ответ
	})
}
