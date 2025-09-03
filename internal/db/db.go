package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Connect(dbURL string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dbURL), &gorm.Config{})
}
