package models

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
)

// PostgresTrainingWorkoutRepository implementa treinos da parte 5.
type PostgresTrainingWorkoutRepository struct {
	db *sql.DB
}

// NewPostgresTrainingWorkoutRepository cria o repositório.
func NewPostgresTrainingWorkoutRepository(db *sql.DB) *PostgresTrainingWorkoutRepository {
	return &PostgresTrainingWorkoutRepository{db: db}
}

// IsMonthInUserActivePlan verifica se o mês pertence ao plano ativo do usuário.
func (r *PostgresTrainingWorkoutRepository) IsMonthInUserActivePlan(userID, monthID uuid.UUID) (bool, error) {
	var ok bool
	err := r.db.QueryRowContext(context.Background(), `
		SELECT EXISTS (
			SELECT 1
			FROM user_training_selection uts
			INNER JOIN training_months tm ON tm.plan_id = uts.plan_id AND tm.id = $2::uuid
			WHERE uts.user_id = $1::uuid AND uts.is_active = true AND tm.is_active = true
		)`, userID, monthID).Scan(&ok)
	return ok, err
}

// ListMonthWorkouts executa get_month_workouts.
func (r *PostgresTrainingWorkoutRepository) ListMonthWorkouts(monthID uuid.UUID) ([]*MonthWorkoutSummary, error) {
	rows, err := r.db.QueryContext(context.Background(),
		`SELECT workout_id, day_number, day_name::text, title, total_exercises FROM get_month_workouts($1::uuid)`,
		monthID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*MonthWorkoutSummary
	for rows.Next() {
		var s MonthWorkoutSummary
		var dayName sql.NullString
		if err := rows.Scan(&s.WorkoutID, &s.DayNumber, &dayName, &s.Title, &s.TotalExercises); err != nil {
			return nil, err
		}
		if dayName.Valid && dayName.String != "" {
			dn := dayName.String
			s.DayName = &dn
		}
		out = append(out, &s)
	}
	return out, rows.Err()
}

// ResolveInProgressMonthID mês em progresso não concluído (mais recente por started_at).
func (r *PostgresTrainingWorkoutRepository) ResolveInProgressMonthID(userID uuid.UUID) (uuid.UUID, error) {
	var mid uuid.UUID
	err := r.db.QueryRowContext(context.Background(), `
		SELECT ump.month_id
		FROM user_month_progress ump
		INNER JOIN user_training_selection uts ON uts.user_id = ump.user_id AND uts.is_active = true
		INNER JOIN training_months tm ON tm.id = ump.month_id AND tm.plan_id = uts.plan_id AND tm.is_active = true
		WHERE ump.user_id = $1::uuid AND ump.is_completed = false
		ORDER BY ump.started_at DESC NULLS LAST
		LIMIT 1
	`, userID).Scan(&mid)
	return mid, err
}

// GetWorkoutOfDayForMonth treino do dia no mês (filtra mês; evita misturar meses do mesmo plano).
func (r *PostgresTrainingWorkoutRepository) GetWorkoutOfDayForMonth(userID uuid.UUID, monthID uuid.UUID, dayNumber int) (*WorkoutDetailResponse, error) {
	var w WorkoutPublic
	var desc sql.NullString
	var dayName sql.NullString
	err := r.db.QueryRowContext(context.Background(), `
		SELECT ww.id, ww.title, ww.description, ww.day_number, ww.day_name::text
		FROM user_training_selection uts
		INNER JOIN training_plans tp ON tp.id = uts.plan_id
		INNER JOIN training_months tm ON tm.plan_id = tp.id AND tm.id = $1::uuid AND tm.is_active = true
		INNER JOIN weekly_workouts ww ON ww.month_id = tm.id AND ww.day_number = $2 AND ww.is_active = true
		WHERE uts.user_id = $3::uuid AND uts.is_active = true
		LIMIT 1
	`, monthID, dayNumber, userID).Scan(&w.ID, &w.Title, &desc, &w.DayNumber, &dayName)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		s := desc.String
		w.Description = &s
	}
	if dayName.Valid && dayName.String != "" {
		s := dayName.String
		w.DayName = &s
	}

	exercises, err := r.listWorkoutExercises(w.ID)
	if err != nil {
		return nil, err
	}
	return &WorkoutDetailResponse{Workout: w, Exercises: exercises}, nil
}

// GetWorkoutDetailForUser detalhe do treino se pertencer ao plano ativo do usuário.
func (r *PostgresTrainingWorkoutRepository) GetWorkoutDetailForUser(userID, workoutID uuid.UUID) (*WorkoutDetailResponse, error) {
	var w WorkoutPublic
	var desc sql.NullString
	var dayName sql.NullString
	err := r.db.QueryRowContext(context.Background(), `
		SELECT ww.id, ww.title, ww.description, ww.day_number, ww.day_name::text
		FROM weekly_workouts ww
		INNER JOIN training_months tm ON tm.id = ww.month_id AND tm.is_active = true
		INNER JOIN user_training_selection uts ON uts.plan_id = tm.plan_id AND uts.user_id = $1::uuid AND uts.is_active = true
		WHERE ww.id = $2::uuid AND ww.is_active = true
	`, userID, workoutID).Scan(&w.ID, &w.Title, &desc, &w.DayNumber, &dayName)
	if err != nil {
		return nil, err
	}
	if desc.Valid {
		s := desc.String
		w.Description = &s
	}
	if dayName.Valid && dayName.String != "" {
		s := dayName.String
		w.DayName = &s
	}
	exercises, err := r.listWorkoutExercises(w.ID)
	if err != nil {
		return nil, err
	}
	return &WorkoutDetailResponse{Workout: w, Exercises: exercises}, nil
}

func (r *PostgresTrainingWorkoutRepository) listWorkoutExercises(workoutID uuid.UUID) ([]*WorkoutExercisePublic, error) {
	rows, err := r.db.QueryContext(context.Background(), `
		SELECT we.id, we.order_position, we.sets, we.reps, we.rest_time, we.instructions, we.notes,
		       e.id, e.name, e.equipment, e.target_muscle, e.video_url, e.video_embed_url, e.instructions, e.thumbnail_url
		FROM workout_exercises we
		INNER JOIN exercises e ON e.id = we.exercise_id AND e.is_active = true
		WHERE we.workout_id = $1::uuid AND we.is_active = true
		ORDER BY we.order_position ASC
	`, workoutID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []*WorkoutExercisePublic
	for rows.Next() {
		var we WorkoutExercisePublic
		var reps, insWE, notes sql.NullString
		var rest sql.NullInt64
		var eq, mus, vid, emb, insE, thumb sql.NullString
		if err := rows.Scan(
			&we.WorkoutExerciseID, &we.OrderPosition, &we.Sets, &reps, &rest, &insWE, &notes,
			&we.Exercise.ID, &we.Exercise.Name, &eq, &mus, &vid, &emb, &insE, &thumb,
		); err != nil {
			return nil, err
		}
		if reps.Valid {
			s := reps.String
			we.Reps = &s
		}
		if rest.Valid {
			v := int(rest.Int64)
			we.RestTimeSeconds = &v
		}
		if insWE.Valid {
			s := insWE.String
			we.Instructions = &s
		}
		if notes.Valid {
			s := notes.String
			we.Notes = &s
		}
		if eq.Valid {
			s := eq.String
			we.Exercise.Equipment = &s
		}
		if mus.Valid {
			s := mus.String
			we.Exercise.TargetMuscle = &s
		}
		if vid.Valid {
			s := vid.String
			we.Exercise.VideoURL = &s
		}
		if emb.Valid {
			s := emb.String
			we.Exercise.VideoEmbedURL = &s
		}
		if insE.Valid {
			s := insE.String
			we.Exercise.Instructions = &s
		}
		if thumb.Valid {
			s := thumb.String
			we.Exercise.ThumbnailURL = &s
		}
		out = append(out, &we)
	}
	return out, rows.Err()
}
