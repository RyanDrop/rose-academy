package models

import "github.com/google/uuid"

// MonthWorkoutSummary linha de get_month_workouts.
type MonthWorkoutSummary struct {
	WorkoutID      uuid.UUID `json:"workout_id"`
	DayNumber      int       `json:"day_number"`
	DayName        *string   `json:"day_name,omitempty"`
	Title          string    `json:"title"`
	TotalExercises int64     `json:"total_exercises"`
}

// WorkoutPublic cabeçalho do treino semanal.
type WorkoutPublic struct {
	ID          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description *string   `json:"description,omitempty"`
	DayNumber   int       `json:"day_number"`
	DayName     *string   `json:"day_name,omitempty"`
}

// WorkoutExercisePublic exercício dentro de um treino (workout_exercises + exercises).
type WorkoutExercisePublic struct {
	WorkoutExerciseID uuid.UUID       `json:"workout_exercise_id"`
	OrderPosition     int             `json:"order_position"`
	Sets              int             `json:"sets"`
	Reps              *string         `json:"reps,omitempty"`
	RestTimeSeconds   *int            `json:"rest_time_seconds,omitempty"`
	Instructions      *string         `json:"instructions,omitempty"`
	Notes             *string         `json:"notes,omitempty"`
	Exercise          ExercisePublic  `json:"exercise"`
}

// WorkoutDetailResponse treino com lista ordenada de exercícios.
type WorkoutDetailResponse struct {
	Workout    WorkoutPublic           `json:"workout"`
	Exercises  []*WorkoutExercisePublic `json:"exercises"`
}

// TrainingWorkoutRepository treinos semanais (parte 5) com checagem de plano ativo do usuário.
type TrainingWorkoutRepository interface {
	IsMonthInUserActivePlan(userID, monthID uuid.UUID) (bool, error)
	ListMonthWorkouts(monthID uuid.UUID) ([]*MonthWorkoutSummary, error)
	ResolveInProgressMonthID(userID uuid.UUID) (uuid.UUID, error)
	GetWorkoutOfDayForMonth(userID uuid.UUID, monthID uuid.UUID, dayNumber int) (*WorkoutDetailResponse, error)
	GetWorkoutDetailForUser(userID, workoutID uuid.UUID) (*WorkoutDetailResponse, error)
}
