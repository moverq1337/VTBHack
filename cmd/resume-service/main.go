package resume_service

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/moverq1337/VTBHack/internal/config"
	"github.com/moverq1337/VTBHack/internal/db"
	"github.com/moverq1337/VTBHack/internal/handlers"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	dbConn, err := db.Connect(cfg.DBURL)
	if err != nil {
		log.Fatal(err)
	}

	r := gin.Default()
	handlers.SetupResumeRoutes(r, dbConn)

	log.Printf("Resume Service запущен на порту %s", cfg.HTTPPort)
	if err := r.Run(cfg.HTTPPort); err != nil {
		log.Fatal(err)
	}
}
