package models

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// PostgresUserRepository implementação do UserRepository usando PostgreSQL (esquema DDL do projeto)
type PostgresUserRepository struct {
	db *sql.DB
}

// NewPostgresUserRepository cria um repositório de usuários
func NewPostgresUserRepository(db *sql.DB) *PostgresUserRepository {
	return &PostgresUserRepository{db: db}
}

// Create insere um novo usuário
func (r *PostgresUserRepository) Create(user *User) error {
	return r.CreateInTx(nil, user)
}

// CreateInTx insere um novo usuário dentro da transação tx. Se tx for nil, usa r.db.
func (r *PostgresUserRepository) CreateInTx(tx *sql.Tx, user *User) error {
	var goalID interface{}
	if user.GoalID != nil {
		goalID = *user.GoalID
	}
	now := time.Now()
	q := `
		INSERT INTO users (
			email, password_hash, full_name, gender, date_birth, weight, height,
			goal_id, cellphone, tax_id, profile_img, is_active, email_verified,
			created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)
		RETURNING id, created_at, updated_at
	`
	args := []interface{}{user.Email, user.PasswordHash, user.FullName, user.Gender, user.DateBirth, user.Weight, user.Height, goalID, nullIfEmpty(user.Cellphone), nullIfEmpty(user.TaxID), user.ProfileImg, user.IsActive, user.EmailVerified, now, now}
	if tx != nil {
		return tx.QueryRow(q, args...).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
	}
	return r.db.QueryRow(q, args...).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

// FindByID retorna um usuário pelo ID
func (r *PostgresUserRepository) FindByID(id uuid.UUID) (*User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, password_hash, full_name, gender, date_birth, weight, height,
			goal_id, COALESCE(cellphone,''), COALESCE(tax_id,''), profile_img, is_active, email_verified,
			created_at, updated_at, last_login_at
		FROM users WHERE id = $1
	`, id)
	return scanUser(row)
}

// FindByEmail retorna um usuário pelo email
func (r *PostgresUserRepository) FindByEmail(email string) (*User, error) {
	row := r.db.QueryRow(`
		SELECT id, email, password_hash, full_name, gender, date_birth, weight, height,
			goal_id, COALESCE(cellphone,''), COALESCE(tax_id,''), profile_img, is_active, email_verified,
			created_at, updated_at, last_login_at
		FROM users WHERE email = $1
	`, email)
	return scanUser(row)
}

// FindByTaxID retorna um usuário pelo CPF/CNPJ (tax_id). Compara apenas dígitos para ignorar formatação.
func (r *PostgresUserRepository) FindByTaxID(taxID string) (*User, error) {
	digits := onlyDigits(taxID)
	if digits == "" {
		return nil, nil
	}
	row := r.db.QueryRow(`
		SELECT id, email, password_hash, full_name, gender, date_birth, weight, height,
			goal_id, COALESCE(cellphone,''), COALESCE(tax_id,''), profile_img, is_active, email_verified,
			created_at, updated_at, last_login_at
		FROM users WHERE REGEXP_REPLACE(tax_id, '[^0-9]', '', 'g') = $1
	`, digits)
	return scanUser(row)
}

// UpdateLastLoginAt atualiza last_login_at
func (r *PostgresUserRepository) UpdateLastLoginAt(id uuid.UUID, at time.Time) error {
	_, err := r.db.Exec(`UPDATE users SET last_login_at = $1, updated_at = $2 WHERE id = $3`, at, at, id)
	return err
}

// UpdateProfile atualiza apenas os campos não nulos em p para o usuário id. updated_at é sempre atualizado.
func (r *PostgresUserRepository) UpdateProfile(id uuid.UUID, p UpdateProfileParams) error {
	sets := []string{"updated_at = $1"}
	args := []interface{}{time.Now()}
	pos := 2
	if p.DateBirth != nil {
		sets = append(sets, fmt.Sprintf("date_birth = $%d", pos))
		args = append(args, *p.DateBirth)
		pos++
	}
	if p.Weight != nil {
		sets = append(sets, fmt.Sprintf("weight = $%d", pos))
		args = append(args, *p.Weight)
		pos++
	}
	if p.Height != nil {
		sets = append(sets, fmt.Sprintf("height = $%d", pos))
		args = append(args, *p.Height)
		pos++
	}
	if p.GoalID != nil {
		sets = append(sets, fmt.Sprintf("goal_id = $%d", pos))
		args = append(args, *p.GoalID)
		pos++
	}
	if p.ProfileImg != nil {
		sets = append(sets, fmt.Sprintf("profile_img = $%d", pos))
		args = append(args, *p.ProfileImg)
		pos++
	}
	if p.Cellphone != nil {
		sets = append(sets, fmt.Sprintf("cellphone = $%d", pos))
		args = append(args, nullIfEmpty(*p.Cellphone))
		pos++
	}
	if p.EmailVerified != nil {
		sets = append(sets, fmt.Sprintf("email_verified = $%d", pos))
		args = append(args, *p.EmailVerified)
		pos++
	}
	if len(sets) == 1 {
		return nil
	}
	args = append(args, id)
	q := fmt.Sprintf("UPDATE users SET %s WHERE id = $%d", strings.Join(sets, ", "), pos)
	_, err := r.db.Exec(q, args...)
	return err
}

func scanUser(row *sql.Row) (*User, error) {
	var u User
	var goalID sql.NullString
	var cellphone, taxID sql.NullString
	var lastLogin sql.NullTime
	err := row.Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.FullName, &u.Gender, &u.DateBirth, &u.Weight, &u.Height,
		&goalID, &cellphone, &taxID, &u.ProfileImg, &u.IsActive, &u.EmailVerified,
		&u.CreatedAt, &u.UpdatedAt, &lastLogin,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if goalID.Valid {
		if id, e := uuid.Parse(goalID.String); e == nil {
			u.GoalID = &id
		}
	}
	if cellphone.Valid {
		u.Cellphone = cellphone.String
	}
	if taxID.Valid {
		u.TaxID = taxID.String
	}
	if lastLogin.Valid {
		u.LastLoginAt = &lastLogin.Time
	}
	return &u, nil
}

func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}

// onlyDigits retorna apenas os dígitos da string (para comparar CPF/CNPJ sem formatação).
func onlyDigits(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}
