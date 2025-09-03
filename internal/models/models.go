package models

import (
	"github.com/google/uuid"
	"time"
)

type Vacancy struct {
	ID               uuid.UUID `gorm:"primaryKey;autoIncrement"`
	Title            string    `gorm:"type:varchar(255)"`
	Requirements     string    `gorm:"type:text"`
	Responsibilities string    `gorm:"type:text"`
	CreatedAt        time.Time
}

type Resume struct {
	ID          uuid.UUID `gorm:"primaryKey;autoIncrement"`
	CandidateID uuid.UUID `gorm:"type:uuid"`
	Text        string    `gorm:"type:text"`
	ParsedData  string    `gorm:"type:jsonb"` // JSONB для хранения парсированных данных
	CreatedAt   time.Time
}

type Interview struct {
	ID          uuid.UUID `gorm:"primaryKey;autoIncrement"`
	CandidateID uuid.UUID `gorm:"type:uuid"`
	Transcript  string    `gorm:"type:text"`
	Score       float64
	Report      string `gorm:"type:jsonb"`
	CreatedAt   time.Time
}
