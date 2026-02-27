package models

import (
	"database/sql"

	"github.com/google/uuid"
)

// PostgresGoalRepository implementação do GoalRepository
type PostgresGoalRepository struct {
	db *sql.DB
}

// NewPostgresGoalRepository cria um repositório de metas
func NewPostgresGoalRepository(db *sql.DB) *PostgresGoalRepository {
	return &PostgresGoalRepository{db: db}
}

// FindAll retorna todas as metas ativas
func (r *PostgresGoalRepository) FindAll() ([]*Goal, error) {
	rows, err := r.db.Query(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM goals WHERE is_active = true ORDER BY name
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*Goal
	for rows.Next() {
		var g Goal
		var desc sql.NullString
		if err := rows.Scan(&g.ID, &g.Name, &desc, &g.IsActive, &g.CreatedAt, &g.UpdatedAt); err != nil {
			return nil, err
		}
		if desc.Valid {
			g.Description = &desc.String
		}
		list = append(list, &g)
	}
	return list, rows.Err()
}

// FindByID retorna uma meta pelo ID
func (r *PostgresGoalRepository) FindByID(id uuid.UUID) (*Goal, error) {
	row := r.db.QueryRow(`
		SELECT id, name, description, is_active, created_at, updated_at
		FROM goals WHERE id = $1
	`, id)
	var g Goal
	var desc sql.NullString
	err := row.Scan(&g.ID, &g.Name, &desc, &g.IsActive, &g.CreatedAt, &g.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		g.Description = &desc.String
	}
	return &g, nil
}
