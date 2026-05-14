package models

import (
	"context"
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// PostgresTrainingProgressRepository chama funções da parte 6.
type PostgresTrainingProgressRepository struct {
	db *sql.DB
}

// NewPostgresTrainingProgressRepository cria o repositório.
func NewPostgresTrainingProgressRepository(db *sql.DB) *PostgresTrainingProgressRepository {
	return &PostgresTrainingProgressRepository{db: db}
}

// CompleteWorkout executa complete_workout.
func (r *PostgresTrainingProgressRepository) CompleteWorkout(userID, workoutID uuid.UUID, weekNumber int, notes *string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(context.Background(),
		`SELECT complete_workout($1::uuid, $2::uuid, $3::integer, $4::text)`,
		userID, workoutID, weekNumber, notes,
	).Scan(&id)
	return id, err
}

// LogExerciseSet executa log_exercise_set.
func (r *PostgresTrainingProgressRepository) LogExerciseSet(userID, completionID, workoutExerciseID uuid.UUID, setNumber int, weight *float64, reps *int, notes *string) (uuid.UUID, error) {
	var id uuid.UUID
	err := r.db.QueryRowContext(context.Background(),
		`SELECT log_exercise_set($1::uuid, $2::uuid, $3::uuid, $4::integer, $5::numeric, $6::integer, $7::text)`,
		userID, completionID, workoutExerciseID, setNumber, nullableFloat(weight), nullableInt(reps), notes,
	).Scan(&id)
	return id, err
}

// LogExerciseSetsBatch grava várias séries na mesma transação (tudo ou nada).
func (r *PostgresTrainingProgressRepository) LogExerciseSetsBatch(userID, completionID uuid.UUID, items []LogExerciseSetRequest) ([]uuid.UUID, error) {
	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	var owned bool
	err = tx.QueryRowContext(ctx, `
		SELECT EXISTS (SELECT 1 FROM user_workout_completion WHERE id = $1::uuid AND user_id = $2::uuid)
	`, completionID, userID).Scan(&owned)
	if err != nil {
		return nil, err
	}
	if !owned {
		return nil, fmt.Errorf("conclusão não encontrada ou não pertence ao usuário")
	}

	ids := make([]uuid.UUID, 0, len(items))
	for i, it := range items {
		if it.WorkoutExerciseID == uuid.Nil {
			return nil, fmt.Errorf("sets[%d].workout_exercise_id obrigatório", i)
		}
		if it.SetNumber < 1 {
			return nil, fmt.Errorf("sets[%d].set_number deve ser >= 1", i)
		}
		ok, err := canUserLogSetTx(ctx, tx, userID, completionID, it.WorkoutExerciseID)
		if err != nil {
			return nil, fmt.Errorf("sets[%d]: %w", i, err)
		}
		if !ok {
			return nil, fmt.Errorf("sets[%d]: exercício inválido para esta conclusão", i)
		}
		var sid uuid.UUID
		err = tx.QueryRowContext(ctx,
			`SELECT log_exercise_set($1::uuid, $2::uuid, $3::uuid, $4::integer, $5::numeric, $6::integer, $7::text)`,
			userID, completionID, it.WorkoutExerciseID, it.SetNumber, nullableFloat(it.WeightUsed), nullableInt(it.RepsCompleted), it.Notes,
		).Scan(&sid)
		if err != nil {
			return nil, fmt.Errorf("sets[%d]: %w", i, err)
		}
		ids = append(ids, sid)
	}

	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return ids, nil
}

func canUserLogSetTx(ctx context.Context, tx *sql.Tx, userID, completionID, workoutExerciseID uuid.UUID) (bool, error) {
	var ok bool
	err := tx.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM user_workout_completion uwc
			INNER JOIN workout_exercises we ON we.workout_id = uwc.workout_id AND we.is_active = true
			WHERE uwc.id = $1::uuid AND uwc.user_id = $2::uuid AND we.id = $3::uuid
		)`, completionID, userID, workoutExerciseID).Scan(&ok)
	return ok, err
}

func nullableFloat(p *float64) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func nullableInt(p *int) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

// CanUserLogSet garante que a conclusão e o exercício pertencem ao mesmo treino do usuário.
func (r *PostgresTrainingProgressRepository) CanUserLogSet(userID, completionID, workoutExerciseID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM user_workout_completion uwc
			INNER JOIN workout_exercises we ON we.workout_id = uwc.workout_id AND we.is_active = true
			WHERE uwc.id = $1::uuid AND uwc.user_id = $2::uuid AND we.id = $3::uuid
		)`, completionID, userID, workoutExerciseID).Scan(&ok)
	return ok, err
}

// ResetMonthProgress executa reset_month_progress.
func (r *PostgresTrainingProgressRepository) ResetMonthProgress(userID, monthID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(context.Background(),
		`SELECT reset_month_progress($1::uuid, $2::uuid)`,
		userID, monthID,
	).Scan(&ok)
	return ok, err
}

// GetUserWeightProgression executa get_user_weight_progression.
func (r *PostgresTrainingProgressRepository) GetUserWeightProgression(userID, monthID uuid.UUID) ([]*WeightProgressionRow, error) {
	rows, err := r.db.QueryContext(context.Background(),
		`SELECT exercise_name, equipment, target_muscle, week_number, set_number, weight_used, reps_completed, completed_at
		 FROM get_user_weight_progression($1::uuid, $2::uuid)`,
		userID, monthID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*WeightProgressionRow
	for rows.Next() {
		var row WeightProgressionRow
		var eq, mus sql.NullString
		var w sql.NullFloat64
		var rep sql.NullInt64
		var at time.Time
		if err := rows.Scan(&row.ExerciseName, &eq, &mus, &row.WeekNumber, &row.SetNumber, &w, &rep, &at); err != nil {
			return nil, err
		}
		if eq.Valid {
			s := eq.String
			row.Equipment = &s
		}
		if mus.Valid {
			s := mus.String
			row.TargetMuscle = &s
		}
		if w.Valid {
			v := w.Float64
			row.WeightUsed = &v
		}
		if rep.Valid {
			v := int(rep.Int64)
			row.RepsCompleted = &v
		}
		row.CompletedAt = at.UTC().Format(time.RFC3339)
		out = append(out, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []*WeightProgressionRow{}
	}
	return out, nil
}

// GetWeeklyBestPerformance executa get_weekly_best_performance.
func (r *PostgresTrainingProgressRepository) GetWeeklyBestPerformance(userID, monthID uuid.UUID) ([]*WeeklyBestPerformanceRow, error) {
	rows, err := r.db.QueryContext(context.Background(),
		`SELECT exercise_id, exercise_name, equipment, target_muscle, week_number, max_weight, avg_weight, total_sets, avg_reps, last_completed
		 FROM get_weekly_best_performance($1::uuid, $2::uuid)`,
		userID, monthID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*WeeklyBestPerformanceRow
	for rows.Next() {
		var row WeeklyBestPerformanceRow
		var eq, mus sql.NullString
		var mw, aw, ar sql.NullFloat64
		var lc time.Time
		if err := rows.Scan(&row.ExerciseID, &row.ExerciseName, &eq, &mus, &row.WeekNumber, &mw, &aw, &row.TotalSets, &ar, &lc); err != nil {
			return nil, err
		}
		if eq.Valid {
			s := eq.String
			row.Equipment = &s
		}
		if mus.Valid {
			s := mus.String
			row.TargetMuscle = &s
		}
		if mw.Valid {
			v := mw.Float64
			row.MaxWeight = &v
		}
		if aw.Valid {
			v := aw.Float64
			row.AvgWeight = &v
		}
		if ar.Valid {
			v := ar.Float64
			row.AvgReps = &v
		}
		row.LastCompleted = lc.UTC().Format(time.RFC3339)
		out = append(out, &row)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if out == nil {
		out = []*WeeklyBestPerformanceRow{}
	}
	return out, nil
}

// GetMonthlyProgressionSummary agrega get_weekly_best_performance em visão mensal (pivot 4 semanas).
// Implementado em Go para evitar ambiguidade de colunas na função SQL get_monthly_progression_summary (RETURNS TABLE vs CTE).
func (r *PostgresTrainingProgressRepository) GetMonthlyProgressionSummary(userID, monthID uuid.UUID) ([]*MonthlyProgressionSummaryRow, error) {
	weekly, err := r.GetWeeklyBestPerformance(userID, monthID)
	if err != nil {
		return nil, err
	}

	type agg struct {
		name       string
		equipment  *string
		targetMus  *string
		weekWeight [5]*float64 // índices 1–4
	}
	byExercise := make(map[uuid.UUID]*agg)
	for _, row := range weekly {
		a, ok := byExercise[row.ExerciseID]
		if !ok {
			a = &agg{
				name:      row.ExerciseName,
				equipment: row.Equipment,
				targetMus: row.TargetMuscle,
			}
			byExercise[row.ExerciseID] = a
		}
		if row.WeekNumber < 1 || row.WeekNumber > 4 || row.MaxWeight == nil {
			continue
		}
		v := *row.MaxWeight
		a.weekWeight[row.WeekNumber] = float64Ptr(v)
	}

	ids := make([]uuid.UUID, 0, len(byExercise))
	for id := range byExercise {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool {
		return byExercise[ids[i]].name < byExercise[ids[j]].name
	})

	out := make([]*MonthlyProgressionSummaryRow, 0, len(ids))
	for _, id := range ids {
		a := byExercise[id]
		w1, w2, w3, w4 := a.weekWeight[1], a.weekWeight[2], a.weekWeight[3], a.weekWeight[4]
		initial := w1
		final := pickLatestWeekWeight(w1, w2, w3, w4)
		row := &MonthlyProgressionSummaryRow{
			ExerciseName: a.name,
			Equipment:    a.equipment,
			TargetMuscle: a.targetMus,
			Week1Weight:  w1,
			Week2Weight:  w2,
			Week3Weight:  w3,
			Week4Weight:  w4,
			InitialWeight: initial,
			FinalWeight:   final,
		}
		if initial != nil && final != nil {
			g := *final - *initial
			row.WeightGain = float64Ptr(g)
			if *initial > 0 {
				pct := (*final - *initial) / *initial * 100
				row.PercentageGain = float64Ptr(math.Round(pct*100) / 100)
			} else {
				z := 0.0
				row.PercentageGain = &z
			}
		}
		out = append(out, row)
	}
	if len(out) == 0 {
		return []*MonthlyProgressionSummaryRow{}, nil
	}
	return out, nil
}

func float64Ptr(v float64) *float64 {
	return &v
}

// pickLatestWeekWeight equivale a COALESCE(w4,w3,w2,w1) no SQL.
func pickLatestWeekWeight(w1, w2, w3, w4 *float64) *float64 {
	for _, p := range []*float64{w4, w3, w2, w1} {
		if p != nil {
			return p
		}
	}
	return nil
}

// GetMonthStatistics executa get_month_statistics (uma linha).
func (r *PostgresTrainingProgressRepository) GetMonthStatistics(userID, monthID uuid.UUID) (*MonthStatisticsRow, error) {
	var row MonthStatisticsRow
	var avgW, maxW, vol, pct sql.NullFloat64
	err := r.db.QueryRowContext(context.Background(),
		`SELECT total_workouts, total_exercises_completed, total_sets_completed, average_weight, max_weight, total_volume, weeks_completed, completion_percentage
		 FROM get_month_statistics($1::uuid, $2::uuid)`,
		userID, monthID,
	).Scan(
		&row.TotalWorkoutsCompleted,
		&row.TotalExercisesCompleted,
		&row.TotalSetsCompleted,
		&avgW,
		&maxW,
		&vol,
		&row.WeeksCompleted,
		&pct,
	)
	if err != nil {
		return nil, err
	}
	if avgW.Valid {
		v := avgW.Float64
		row.AverageWeight = &v
	}
	if maxW.Valid {
		v := maxW.Float64
		row.MaxWeight = &v
	}
	if vol.Valid {
		v := vol.Float64
		row.TotalVolume = &v
	}
	if pct.Valid {
		v := pct.Float64
		row.CompletionPercentage = &v
	}
	return &row, nil
}

// WorkoutAccessibleForProgress indica se o treino pertence ao plano ativo do usuário (para registrar conclusão).
func (r *PostgresTrainingProgressRepository) WorkoutAccessibleForProgress(userID, workoutID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM weekly_workouts ww
			INNER JOIN training_months tm ON tm.id = ww.month_id AND tm.is_active = true
			INNER JOIN user_training_selection uts ON uts.plan_id = tm.plan_id AND uts.user_id = $1::uuid AND uts.is_active = true
			WHERE ww.id = $2::uuid AND ww.is_active = true
		)`, userID, workoutID).Scan(&ok)
	return ok, err
}
