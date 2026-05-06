package models

import "github.com/google/uuid"

// TrainingMonthPublic mês de treino na API (sem is_active nem timestamps).
type TrainingMonthPublic struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Subtitle    *string   `json:"subtitle,omitempty"` // description no banco
	MonthNumber int       `json:"month_number"`
}

// TrainingPlanPublic plano com meses aninhados (sem is_active nem timestamps).
// title vem da coluna name; subtitle de description.
type TrainingPlanPublic struct {
	ID       uuid.UUID              `json:"id"`
	Title    string                 `json:"title"` // name
	Subtitle *string                `json:"subtitle,omitempty"`
	Level    *string                `json:"level,omitempty"` // PT-BR canônico (Iniciante, Intermediário, Avançado) mesmo que o banco guarde sem acentos
	Months   []*TrainingMonthPublic `json:"months"`
}

// TrainingPlanRepository leitura de planos e meses de treino.
type TrainingPlanRepository interface {
	ListActivePlansWithMonths() ([]*TrainingPlanPublic, error)
}
