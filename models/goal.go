package models

import (
	"time"

	"github.com/google/uuid"
)

// Goal representa a entidade meta (tabela goals)
type Goal struct {
	ID          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// GoalRepository define as operações de persistência para Goal
type GoalRepository interface {
	FindAll() ([]*Goal, error)
	FindByID(id uuid.UUID) (*Goal, error)
}
