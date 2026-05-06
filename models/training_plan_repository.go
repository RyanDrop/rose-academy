package models

import (
	"database/sql"
	"sort"
	"strings"

	"github.com/google/uuid"
)

// PostgresTrainingPlanRepository implementação em PostgreSQL.
type PostgresTrainingPlanRepository struct {
	db *sql.DB
}

// NewPostgresTrainingPlanRepository cria o repositório.
func NewPostgresTrainingPlanRepository(db *sql.DB) *PostgresTrainingPlanRepository {
	return &PostgresTrainingPlanRepository{db: db}
}

// ListActivePlansWithMonths retorna planos ativos com meses ativos, ordenados.
func (r *PostgresTrainingPlanRepository) ListActivePlansWithMonths() ([]*TrainingPlanPublic, error) {
	planRows, err := r.db.Query(`
		SELECT id, name, description, level::text AS level_text
		FROM training_plans
		WHERE is_active = true
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer planRows.Close()

	byID := make(map[uuid.UUID]*TrainingPlanPublic)
	order := make([]uuid.UUID, 0)

	for planRows.Next() {
		var id uuid.UUID
		var name string
		var desc, levelText sql.NullString
		if err := planRows.Scan(&id, &name, &desc, &levelText); err != nil {
			return nil, err
		}
		pl := &TrainingPlanPublic{
			ID:     id,
			Title:  name,
			Months: []*TrainingMonthPublic{},
		}
		if desc.Valid {
			s := desc.String
			pl.Subtitle = &s
		}
		if levelText.Valid && levelText.String != "" {
			s := formatTrainingLevelDisplay(levelText.String)
			pl.Level = &s
		}
		byID[id] = pl
		order = append(order, id)
	}
	if err := planRows.Err(); err != nil {
		return nil, err
	}

	monthRows, err := r.db.Query(`
		SELECT m.id, m.plan_id, m.month_number, m.title, m.description
		FROM training_months m
		INNER JOIN training_plans p ON p.id = m.plan_id AND p.is_active = true
		WHERE m.is_active = true
		ORDER BY p.name ASC, m.month_number ASC, m.title ASC
	`)
	if err != nil {
		return nil, err
	}
	defer monthRows.Close()

	for monthRows.Next() {
		var mid, planID uuid.UUID
		var mTitle string
		var mDesc sql.NullString
		var monthNum int
		if err := monthRows.Scan(&mid, &planID, &monthNum, &mTitle, &mDesc); err != nil {
			return nil, err
		}
		pl, ok := byID[planID]
		if !ok {
			continue
		}
		month := &TrainingMonthPublic{
			ID:          mid,
			Title:       mTitle,
			MonthNumber: monthNum,
		}
		if mDesc.Valid {
			s := mDesc.String
			month.Subtitle = &s
		}
		pl.Months = append(pl.Months, month)
	}
	if err := monthRows.Err(); err != nil {
		return nil, err
	}

	out := make([]*TrainingPlanPublic, 0, len(order))
	for _, id := range order {
		out = append(out, byID[id])
	}
	sort.SliceStable(out, func(i, j int) bool {
		ri, rj := trainingLevelRank(out[i].Level), trainingLevelRank(out[j].Level)
		if ri != rj {
			return ri < rj
		}
		return strings.Compare(out[i].Title, out[j].Title) < 0
	})
	return out, nil
}
