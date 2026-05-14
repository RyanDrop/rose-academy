package models

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
)

// PostgresUserTrainingRepository chama funções PostgreSQL da parte 4.
type PostgresUserTrainingRepository struct {
	db *sql.DB
}

// NewPostgresUserTrainingRepository cria o repositório.
func NewPostgresUserTrainingRepository(db *sql.DB) *PostgresUserTrainingRepository {
	return &PostgresUserTrainingRepository{db: db}
}

// SelectTrainingPlan executa select_training_plan.
func (r *PostgresUserTrainingRepository) SelectTrainingPlan(userID uuid.UUID, levelDBEnum string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(context.Background(),
		`SELECT select_training_plan($1::uuid, $2::training_level_type)`,
		userID, levelDBEnum,
	).Scan(&id)
	return id, err
}

// GetActivePlan executa get_user_active_plan (no máximo uma linha).
func (r *PostgresUserTrainingRepository) GetActivePlan(userID uuid.UUID) (*UserActivePlanResponse, error) {
	var planID uuid.UUID
	var levelRaw, planName string
	var selectedAt time.Time
	err := r.db.QueryRowContext(context.Background(),
		`SELECT plan_id, plan_level::text, plan_name, selected_at FROM get_user_active_plan($1::uuid)`,
		userID,
	).Scan(&planID, &levelRaw, &planName, &selectedAt)
	if err != nil {
		return nil, err
	}
	return &UserActivePlanResponse{
		PlanID:     planID,
		Level:      formatTrainingLevelDisplay(levelRaw),
		PlanName:   planName,
		SelectedAt: selectedAt,
	}, nil
}

// ListPlanMonths executa get_user_plan_months.
func (r *PostgresUserTrainingRepository) ListPlanMonths(userID uuid.UUID) ([]*UserTrainingMonthItem, error) {
	rows, err := r.db.QueryContext(context.Background(),
		`SELECT month_id, month_number, month_title, month_description, is_started, is_completed, started_at, completed_at
		 FROM get_user_plan_months($1::uuid)`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*UserTrainingMonthItem
	for rows.Next() {
		var it UserTrainingMonthItem
		var desc sql.NullString
		var startedAt, completedAt sql.NullTime
		if err := rows.Scan(
			&it.MonthID,
			&it.MonthNumber,
			&it.Title,
			&desc,
			&it.IsStarted,
			&it.IsCompleted,
			&startedAt,
			&completedAt,
		); err != nil {
			return nil, err
		}
		if desc.Valid {
			s := desc.String
			it.Description = &s
		}
		if startedAt.Valid {
			t := startedAt.Time
			it.StartedAt = &t
		}
		if completedAt.Valid {
			t := completedAt.Time
			it.CompletedAt = &t
		}
		out = append(out, &it)
	}
	return out, rows.Err()
}

// HasActiveTrainingSelection indica se existe user_training_selection ativa.
func (r *PostgresUserTrainingRepository) HasActiveTrainingSelection(userID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(context.Background(),
		`SELECT EXISTS (
			SELECT 1 FROM user_training_selection WHERE user_id = $1::uuid AND is_active = true
		)`,
		userID,
	).Scan(&ok)
	return ok, err
}

// HasUserStartedAnyTrainingMonth indica se algum mês do plano ativo está iniciado (via get_user_plan_months, alinhado a GET /me/training/months).
func (r *PostgresUserTrainingRepository) HasUserStartedAnyTrainingMonth(userID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(context.Background(),
		`SELECT EXISTS (
			SELECT 1 FROM get_user_plan_months($1::uuid)
			WHERE is_started IS TRUE
		)`,
		userID,
	).Scan(&ok)
	return ok, err
}

// StartMonthProgress executa start_month_progress.
func (r *PostgresUserTrainingRepository) StartMonthProgress(userID uuid.UUID, monthNumber int) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(context.Background(),
		`SELECT start_month_progress($1::uuid, $2::integer)`,
		userID, monthNumber,
	).Scan(&id)
	return id, err
}

// IsPGRaiseException detecta RAISE EXCEPTION em plpgsql (código P0001).
func IsPGRaiseException(err error) bool {
	var pe *pgconn.PgError
	if !errors.As(err, &pe) {
		return false
	}
	return pe.Code == "P0001"
}
