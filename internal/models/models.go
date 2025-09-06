package models

import (
	"github.com/google/uuid"
	"time"
)

type Vacancy struct {
	ID               uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	Title            string    `gorm:"type:varchar(255)"`
	Requirements     string    `gorm:"type:text"`
	Responsibilities string    `gorm:"type:text"`
	Region           string    `gorm:"type:varchar(100)"`
	City             string    `gorm:"type:varchar(100)"`
	EmploymentType   string    `gorm:"type:varchar(50)"`  // Полная, частичная, удаленная
	WorkSchedule     string    `gorm:"type:varchar(50)"`  // Полный день, сменный и т.д.
	Experience       string    `gorm:"type:varchar(50)"`  // Требуемый опыт
	Education        string    `gorm:"type:varchar(100)"` // Требуемое образование
	SalaryMin        int       `gorm:"type:integer"`
	SalaryMax        int       `gorm:"type:integer"`
	Languages        string    `gorm:"type:text"` // Требуемые языки
	Skills           string    `gorm:"type:text"` // Ключевые навыки
	CreatedAt        time.Time
}

type Resume struct {
	ID           uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	CandidateID  uuid.UUID `gorm:"type:uuid"`
	Text         string    `gorm:"type:text"`
	ParsedData   string    `gorm:"type:jsonb"`
	FileURL      string    `gorm:"type:text"`
	Experience   int       `gorm:"type:integer"` // Опыт в годах
	Education    string    `gorm:"type:varchar(100)"`
	Skills       string    `gorm:"type:text"`
	Languages    string    `gorm:"type:text"`
	SalaryExpect int       `gorm:"type:integer"`
	CreatedAt    time.Time
}

type AnalysisResult struct {
	ID         uuid.UUID `gorm:"primaryKey;type:uuid;default:uuid_generate_v4()"`
	ResumeID   uuid.UUID `gorm:"type:uuid"`
	VacancyID  uuid.UUID `gorm:"type:uuid"`
	MatchScore float64   `gorm:"type:decimal(5,2)"`
	Details    string    `gorm:"type:jsonb"` // Детали анализа
	CreatedAt  time.Time
}
