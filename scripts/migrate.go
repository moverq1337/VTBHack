package main

import (
	"log"

	"github.com/moverq1337/VTBHack/internal/config"
	"github.com/moverq1337/VTBHack/internal/db"
	"github.com/moverq1337/VTBHack/internal/models"
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

	err = dbConn.AutoMigrate(&models.Vacancy{}, &models.Resume{}, &models.Interview{})
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Миграции завершены")
}
