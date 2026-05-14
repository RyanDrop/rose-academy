package models

import "github.com/google/uuid"

// CompleteWorkoutResponse retorno de complete_workout.
type CompleteWorkoutResponse struct {
	CompletionID uuid.UUID `json:"completion_id"`
}

// LogExerciseSetResponse retorno de log_exercise_set.
type LogExerciseSetResponse struct {
	SetID uuid.UUID `json:"set_id"`
}

// ResetMonthProgressResponse retorno de reset_month_progress.
type ResetMonthProgressResponse struct {
	Reset bool `json:"reset"`
}

// WeightProgressionRow linha de get_user_weight_progression.
type WeightProgressionRow struct {
	ExerciseName   string     `json:"exercise_name"`
	Equipment      *string    `json:"equipment,omitempty"`
	TargetMuscle   *string    `json:"target_muscle,omitempty"`
	WeekNumber     int        `json:"week_number"`
	SetNumber      int        `json:"set_number"`
	WeightUsed     *float64   `json:"weight_used,omitempty"`
	RepsCompleted  *int       `json:"reps_completed,omitempty"`
	CompletedAt    string     `json:"completed_at"` // RFC3339
}

// WeeklyBestPerformanceRow linha de get_weekly_best_performance.
type WeeklyBestPerformanceRow struct {
	ExerciseID     uuid.UUID  `json:"exercise_id"`
	ExerciseName   string     `json:"exercise_name"`
	Equipment      *string    `json:"equipment,omitempty"`
	TargetMuscle   *string    `json:"target_muscle,omitempty"`
	WeekNumber     int        `json:"week_number"`
	MaxWeight      *float64   `json:"max_weight,omitempty"`
	AvgWeight      *float64   `json:"avg_weight,omitempty"`
	TotalSets      int        `json:"total_sets"`
	AvgReps        *float64   `json:"avg_reps,omitempty"`
	LastCompleted  string     `json:"last_completed"` // RFC3339
}

// MonthlyProgressionSummaryRow linha de get_monthly_progression_summary.
type MonthlyProgressionSummaryRow struct {
	ExerciseName     string    `json:"exercise_name"`
	Equipment          *string   `json:"equipment,omitempty"`
	TargetMuscle       *string   `json:"target_muscle,omitempty"`
	Week1Weight        *float64  `json:"week1_weight,omitempty"`
	Week2Weight        *float64  `json:"week2_weight,omitempty"`
	Week3Weight        *float64  `json:"week3_weight,omitempty"`
	Week4Weight        *float64  `json:"week4_weight,omitempty"`
	InitialWeight      *float64  `json:"initial_weight,omitempty"`
	FinalWeight        *float64  `json:"final_weight,omitempty"`
	WeightGain         *float64  `json:"weight_gain,omitempty"`
	PercentageGain     *float64  `json:"percentage_gain,omitempty"`
}

// MonthStatisticsRow linha única de get_month_statistics.
type MonthStatisticsRow struct {
	TotalWorkoutsCompleted       int      `json:"total_workouts_completed"`
	TotalExercisesCompleted      int      `json:"total_exercises_completed"`
	TotalSetsCompleted           int      `json:"total_sets_completed"`
	AverageWeight                *float64 `json:"average_weight,omitempty"`
	MaxWeight                    *float64 `json:"max_weight,omitempty"`
	TotalVolume                  *float64 `json:"total_volume,omitempty"`
	WeeksCompleted               int      `json:"weeks_completed"`
	CompletionPercentage         *float64 `json:"completion_percentage,omitempty"`
}

// LogExerciseSetRequest body POST .../sets.
type LogExerciseSetRequest struct {
	WorkoutExerciseID uuid.UUID  `json:"workout_exercise_id"`
	SetNumber         int        `json:"set_number"`
	WeightUsed        *float64   `json:"weight_used,omitempty"`
	RepsCompleted     *int       `json:"reps_completed,omitempty"`
	Notes             *string    `json:"notes,omitempty"`
}

// LogExerciseSetsBatchRequest body POST .../sets/batch (todas as séries ao fim do treino).
type LogExerciseSetsBatchRequest struct {
	Sets []LogExerciseSetRequest `json:"sets"`
}

// LogExerciseSetsBatchResponse retorno do lote de séries.
type LogExerciseSetsBatchResponse struct {
	SetIDs []uuid.UUID `json:"set_ids"`
	Count  int         `json:"count"`
}

// CompleteWorkoutRequest body POST .../complete.
type CompleteWorkoutRequest struct {
	WeekNumber int     `json:"week_number"`
	Notes      *string `json:"notes,omitempty"`
}

// TrainingProgressRepository funções SQL da parte 6.
type TrainingProgressRepository interface {
	CompleteWorkout(userID, workoutID uuid.UUID, weekNumber int, notes *string) (uuid.UUID, error)
	LogExerciseSet(userID, completionID, workoutExerciseID uuid.UUID, setNumber int, weight *float64, reps *int, notes *string) (uuid.UUID, error)
	LogExerciseSetsBatch(userID, completionID uuid.UUID, items []LogExerciseSetRequest) ([]uuid.UUID, error)
	CanUserLogSet(userID, completionID, workoutExerciseID uuid.UUID) (bool, error)
	ResetMonthProgress(userID, monthID uuid.UUID) (bool, error)
	GetUserWeightProgression(userID, monthID uuid.UUID) ([]*WeightProgressionRow, error)
	GetWeeklyBestPerformance(userID, monthID uuid.UUID) ([]*WeeklyBestPerformanceRow, error)
	GetMonthlyProgressionSummary(userID, monthID uuid.UUID) ([]*MonthlyProgressionSummaryRow, error)
	GetMonthStatistics(userID, monthID uuid.UUID) (*MonthStatisticsRow, error)
	WorkoutAccessibleForProgress(userID, workoutID uuid.UUID) (bool, error)
}
