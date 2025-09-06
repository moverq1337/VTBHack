package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/moverq1337/VTBHack/internal/config"
	"github.com/moverq1337/VTBHack/internal/db"
	"github.com/moverq1337/VTBHack/internal/handlers"
	"github.com/moverq1337/VTBHack/scripts"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	scripts.Migrate()

	dbConn, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()

	// Настройка CORS для фронтенда
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Настройка маршрутов API Gateway
	handlers.SetupRoutes(r, dbConn)

	log.Printf("API Gateway запущен на порту %s", cfg.HTTPPort)
	if err := r.Run(cfg.HTTPPort); err != nil {
		log.Fatal(err)
	}
}
