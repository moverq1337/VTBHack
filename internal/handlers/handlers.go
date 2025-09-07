package handlers

import (
	"os"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// SetupRoutes настраивает маршруты для API Gateway
func SetupRoutes(r *gin.Engine, db *gorm.DB) {
	api := r.Group("/api")
	{
		api.POST("/upload/resume", func(c *gin.Context) { UploadResume(c, db) })
		api.POST("/upload/vacancy", func(c *gin.Context) { UploadVacancy(c, db) })
		api.POST("/analyze", func(c *gin.Context) { AnalyzeResume(c, db) })
		api.GET("/health", HealthCheck)
	}

	r.GET("/interview", func(c *gin.Context) {
		// Логируем попытку доступа к файлу
		log.Printf("Serving interview page")

		// Проверяем существование файла
		if _, err := os.Stat("/app/frontend/interview.html"); os.IsNotExist(err) {
			log.Printf("File does not exist: %v", err)
			c.JSON(404, gin.H{"error": "File not found"})
			return
		}

		c.File("/app/frontend/interview.html")
	})

	r.GET("/health", HealthCheck)
}

// SetupResumeRoutes настраивает маршруты для Resume Service
func SetupResumeRoutes(r *gin.Engine, db *gorm.DB) {
	r.POST("/upload/resume", func(c *gin.Context) { UploadResume(c, db) })
	r.POST("/upload/vacancy", func(c *gin.Context) { UploadVacancy(c, db) })
	r.POST("/analyze", func(c *gin.Context) { AnalyzeResume(c, db) })
	r.GET("/health", HealthCheck)
}

// HealthCheck проверяет статус сервиса
func HealthCheck(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}
