package models

import (
	"time"

	"github.com/google/uuid"
)

type Vacancy struct {
	ID               uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Title            string    `gorm:"type:varchar(255)"`
	Requirements     string    `gorm:"type:text"`
	Responsibilities string    `gorm:"type:text"`
	CreatedAt        time.Time
}

type Resume struct {
	ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	CandidateID uuid.UUID `gorm:"type:uuid"`
	Text        string    `gorm:"type:text"`
	ParsedData  string    `gorm:"type:jsonb"`
	FileURL     string    `gorm:"type:text"`
	CreatedAt   time.Time
}

type Interview struct {
	ID          uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	CandidateID uuid.UUID `gorm:"type:uuid"`
	VacancyID   uuid.UUID `gorm:"type:uuid"`
	Transcript  string    `gorm:"type:text"`
	Score       float64
	Report      string `gorm:"type:jsonb"`
	Status      string `gorm:"type:varchar(50)"`
	CreatedAt   time.Time
	CompletedAt *time.Time
}
