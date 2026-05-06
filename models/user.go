package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

// GenderType valores do enum public.gender_type
const (
	GenderFeminino         = "feminino"
	GenderMasculino        = "masculino"
	GenderOutro            = "outro"
	GenderPrefiroNaoDizer  = "prefiro_nao_dizer"
)

// User representa a entidade usuário (tabela users do DDL)
type User struct {
	ID            uuid.UUID  `json:"id"`
	Email         string     `json:"email"`
	PasswordHash  string     `json:"-"`
	FullName      string     `json:"full_name"`
	Gender        *string    `json:"gender,omitempty"`
	DateBirth     *time.Time `json:"date_birth,omitempty"`
	Weight        *float64   `json:"weight,omitempty"`
	Height        *float64   `json:"height,omitempty"`
	GoalID        *uuid.UUID `json:"goal_id,omitempty"`
	Cellphone     string     `json:"cellphone"`
	TaxID         string     `json:"tax_id"`
	ProfileImg    *string    `json:"profile_img,omitempty"`
	IsActive      bool       `json:"is_active"`
	EmailVerified bool       `json:"email_verified"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
	LastLoginAt   *time.Time `json:"last_login_at,omitempty"`
}

// UpdateProfileParams campos opcionais para PATCH do perfil (apenas campos presentes são atualizados)
type UpdateProfileParams struct {
	DateBirth     *time.Time
	Weight        *float64
	Height        *float64
	GoalID        *uuid.UUID
	ProfileImg    *string
	Cellphone     *string
	TaxID         *string // apenas dígitos; string vazia zera tax_id no banco
	Gender        *string // valores gender_type; string vazia zera gender
	EmailVerified *bool
}

// UserRepository define as operações de persistência para User
type UserRepository interface {
	Create(user *User) error
	CreateInTx(tx *sql.Tx, user *User) error
	FindByID(id uuid.UUID) (*User, error)
	FindByEmail(email string) (*User, error)
	FindByTaxID(taxID string) (*User, error)
	UpdateLastLoginAt(id uuid.UUID, at time.Time) error
	UpdateProfile(id uuid.UUID, p UpdateProfileParams) error
}
