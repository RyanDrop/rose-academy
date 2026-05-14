package models

import (
	"time"

	"github.com/google/uuid"
)

// UserActivePlanResponse plano ativo do usuário (get_user_active_plan).
type UserActivePlanResponse struct {
	PlanID     uuid.UUID `json:"plan_id"`
	Level      string    `json:"level"` // PT-BR canônico (Iniciante, …)
	PlanName   string    `json:"plan_name"`
	SelectedAt time.Time `json:"selected_at"`
}

// UserTrainingMonthItem mês do plano ativo com progresso (get_user_plan_months).
// Active é calculado na API: sem nenhum mês iniciado (get_user_plan_months.is_started), só mês 1 true;
// depois selected_at + meses de calendário (1..12).
type UserTrainingMonthItem struct {
	MonthID          uuid.UUID  `json:"month_id"`
	MonthNumber      int        `json:"month_number"`
	Title            string     `json:"title"`
	Description      *string    `json:"description,omitempty"`
	IsStarted        bool       `json:"is_started"`
	IsCompleted      bool       `json:"is_completed"`
	StartedAt        *time.Time `json:"started_at,omitempty"`
	CompletedAt      *time.Time `json:"completed_at,omitempty"`
	Active           bool       `json:"active"`
}

// SelectTrainingPlanResponse retorno de POST /me/training/plan.
type SelectTrainingPlanResponse struct {
	SelectionID uuid.UUID `json:"selection_id"`
}

// StartMonthProgressResponse retorno de POST …/months/{n}/start.
type StartMonthProgressResponse struct {
	ProgressID uuid.UUID `json:"progress_id"`
}

// UserTrainingRepository operações de plano/mês do usuário (funções SQL parte 4).
type UserTrainingRepository interface {
	SelectTrainingPlan(userID uuid.UUID, levelDBEnum string) (selectionID uuid.UUID, err error)
	GetActivePlan(userID uuid.UUID) (*UserActivePlanResponse, error)
	ListPlanMonths(userID uuid.UUID) ([]*UserTrainingMonthItem, error)
	HasActiveTrainingSelection(userID uuid.UUID) (bool, error)
	// HasUserStartedAnyTrainingMonth indica se o utilizador já iniciou algum mês (get_user_plan_months, is_started).
	HasUserStartedAnyTrainingMonth(userID uuid.UUID) (bool, error)
	StartMonthProgress(userID uuid.UUID, monthNumber int) (progressID uuid.UUID, err error)
}
