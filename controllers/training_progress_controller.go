package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"rose-academy/api"
	"rose-academy/middleware"
	"rose-academy/models"
)

// TrainingProgressController conclusão de treinos, séries e relatórios (parte 6).
type TrainingProgressController struct {
	progress models.TrainingProgressRepository
	months   models.MonthAccessChecker
	devMode  bool
}

// NewTrainingProgressController cria o controller.
func NewTrainingProgressController(progress models.TrainingProgressRepository, months models.MonthAccessChecker, devMode bool) *TrainingProgressController {
	return &TrainingProgressController{progress: progress, months: months, devMode: devMode}
}

func (c *TrainingProgressController) requireMonthInPlan(w http.ResponseWriter, r *http.Request, userID, monthID uuid.UUID) bool {
	ok, err := c.months.IsMonthInUserActivePlan(userID, monthID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return false
	}
	if !ok {
		api.Write(w, r, http.StatusForbidden, "Forbidden", "este mês não pertence ao seu plano ativo")
		return false
	}
	return true
}

// CompleteWorkout registra conclusão (complete_workout).
// @Summary		Concluir treino
// @Description	week_number de 1 a 4 (semana do mês). Exige progresso no mês do treino não concluído.
// @Tags			training
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param		workout_id	path	string	true	"UUID weekly_workouts"
// @Param		body	body	models.CompleteWorkoutRequest	true	"week_number, notes opcional"
// @Success		200	{object}	models.CompleteWorkoutResponse
// @Router		/me/training/workouts/{workout_id}/complete [post]
func (c *TrainingProgressController) CompleteWorkout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	workoutID, err := uuid.Parse(chi.URLParam(r, "workout_id"))
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "workout_id inválido")
		return
	}
	var req models.CompleteWorkoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	if req.WeekNumber < 1 || req.WeekNumber > 4 {
		api.WriteValidation(w, r, []api.FieldError{{Field: "week_number", Message: "deve ser entre 1 e 4"}})
		return
	}
	allowed, err := c.progress.WorkoutAccessibleForProgress(userID, workoutID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if !allowed {
		api.Write(w, r, http.StatusForbidden, "Forbidden", "treino não disponível no seu plano ativo")
		return
	}
	completionID, err := c.progress.CompleteWorkout(userID, workoutID, req.WeekNumber, req.Notes)
	if err != nil {
		if models.IsPGRaiseException(err) || strings.Contains(strings.ToLower(err.Error()), "não encontrado") {
			api.Write(w, r, http.StatusConflict, "Conflict", "não foi possível registrar: verifique se o mês está em progresso e não concluído")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(models.CompleteWorkoutResponse{CompletionID: completionID})
}

// LogExerciseSet registra uma série (log_exercise_set).
// @Summary		Registrar série
// @Tags			training
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param		completion_id	path	string	true	"UUID user_workout_completion"
// @Param		body	body	models.LogExerciseSetRequest	true	"workout_exercise_id, set_number, opcionais weight_used, reps_completed, notes"
// @Success		200	{object}	models.LogExerciseSetResponse
// @Router		/me/training/workout-completions/{completion_id}/sets [post]
func (c *TrainingProgressController) LogExerciseSet(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	completionID, err := uuid.Parse(chi.URLParam(r, "completion_id"))
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "completion_id inválido")
		return
	}
	var req models.LogExerciseSetRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	if req.WorkoutExerciseID == uuid.Nil {
		api.WriteValidation(w, r, []api.FieldError{{Field: "workout_exercise_id", Message: "obrigatório"}})
		return
	}
	if req.SetNumber < 1 {
		api.WriteValidation(w, r, []api.FieldError{{Field: "set_number", Message: "deve ser >= 1"}})
		return
	}
	can, err := c.progress.CanUserLogSet(userID, completionID, req.WorkoutExerciseID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if !can {
		api.Write(w, r, http.StatusForbidden, "Forbidden", "conclusão ou exercício inválido para seu usuário")
		return
	}
	setID, err := c.progress.LogExerciseSet(userID, completionID, req.WorkoutExerciseID, req.SetNumber, req.WeightUsed, req.RepsCompleted, req.Notes)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(models.LogExerciseSetResponse{SetID: setID})
}

// LogExerciseSetsBatch grava todas as séries de uma vez (transação única).
// @Summary		Registrar séries em lote
// @Description	Ao fim do treino, envie o array `sets` com workout_exercise_id, set_number e opcionais weight_used, reps_completed, notes.
// @Tags			training
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param		completion_id	path	string	true	"UUID user_workout_completion"
// @Param		body	body	models.LogExerciseSetsBatchRequest	true	"sets: array de LogExerciseSetRequest"
// @Success		200	{object}	models.LogExerciseSetsBatchResponse
// @Router		/me/training/workout-completions/{completion_id}/sets/batch [post]
func (c *TrainingProgressController) LogExerciseSetsBatch(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	completionID, err := uuid.Parse(chi.URLParam(r, "completion_id"))
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "completion_id inválido")
		return
	}
	var req models.LogExerciseSetsBatchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	if len(req.Sets) == 0 {
		api.WriteValidation(w, r, []api.FieldError{{Field: "sets", Message: "envie pelo menos uma série"}})
		return
	}
	var ferrs []api.FieldError
	for i, s := range req.Sets {
		prefix := fmt.Sprintf("sets[%d]", i)
		if s.WorkoutExerciseID == uuid.Nil {
			ferrs = append(ferrs, api.FieldError{Field: prefix + ".workout_exercise_id", Message: "obrigatório"})
		}
		if s.SetNumber < 1 {
			ferrs = append(ferrs, api.FieldError{Field: prefix + ".set_number", Message: "deve ser >= 1"})
		}
	}
	if len(ferrs) > 0 {
		api.WriteValidation(w, r, ferrs)
		return
	}
	ids, err := c.progress.LogExerciseSetsBatch(userID, completionID, req.Sets)
	if err != nil {
		msg := err.Error()
		low := strings.ToLower(msg)
		if strings.Contains(low, "não encontrada") || strings.Contains(low, "não pertence ao usuário") {
			api.Write(w, r, http.StatusForbidden, "Forbidden", "conclusão inválida para seu usuário")
			return
		}
		if strings.Contains(msg, "inválido para esta conclusão") {
			api.Write(w, r, http.StatusForbidden, "Forbidden", "um ou mais exercícios não pertencem a este treino")
			return
		}
		if strings.Contains(msg, "sets[") {
			api.Write(w, r, http.StatusBadRequest, "Bad Request", msg)
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(models.LogExerciseSetsBatchResponse{SetIDs: ids, Count: len(ids)})
}

// ResetMonthProgress apaga conclusões e séries do mês (reset_month_progress).
// @Summary		Resetar progresso do mês
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		month_id	path	string	true	"UUID training_months"
// @Success		200	{object}	models.ResetMonthProgressResponse
// @Router		/me/training/months/{month_id}/progress/reset [post]
func (c *TrainingProgressController) ResetMonthProgress(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	monthID, err := uuid.Parse(chi.URLParam(r, "month_id"))
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "month_id inválido")
		return
	}
	if !c.requireMonthInPlan(w, r, userID, monthID) {
		return
	}
	did, err := c.progress.ResetMonthProgress(userID, monthID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if !did {
		api.Write(w, r, http.StatusNotFound, "Not Found", "nenhum progresso encontrado para este mês")
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(models.ResetMonthProgressResponse{Reset: true})
}

// GetWeightProgression GET get_user_weight_progression.
// @Summary		Progressão de cargas no mês
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		month_id	path	string	true	"UUID training_months"
// @Success		200	{array}	models.WeightProgressionRow
// @Router		/me/training/months/{month_id}/weight-progression [get]
func (c *TrainingProgressController) GetWeightProgression(w http.ResponseWriter, r *http.Request) {
	c.getMonthReport(w, r, func(userID, monthID uuid.UUID) (interface{}, error) {
		return c.progress.GetUserWeightProgression(userID, monthID)
	})
}

// GetWeeklyBestPerformance GET get_weekly_best_performance.
// @Summary		Melhor desempenho semanal por exercício
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		month_id	path	string	true	"UUID training_months"
// @Success		200	{array}	models.WeeklyBestPerformanceRow
// @Router		/me/training/months/{month_id}/weekly-performance [get]
func (c *TrainingProgressController) GetWeeklyBestPerformance(w http.ResponseWriter, r *http.Request) {
	c.getMonthReport(w, r, func(userID, monthID uuid.UUID) (interface{}, error) {
		return c.progress.GetWeeklyBestPerformance(userID, monthID)
	})
}

// GetMonthlyProgressionSummary GET get_monthly_progression_summary.
// @Summary		Resumo comparativo mensal por exercício
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		month_id	path	string	true	"UUID training_months"
// @Success		200	{array}	models.MonthlyProgressionSummaryRow
// @Router		/me/training/months/{month_id}/progression-summary [get]
func (c *TrainingProgressController) GetMonthlyProgressionSummary(w http.ResponseWriter, r *http.Request) {
	c.getMonthReport(w, r, func(userID, monthID uuid.UUID) (interface{}, error) {
		return c.progress.GetMonthlyProgressionSummary(userID, monthID)
	})
}

// GetMonthStatistics GET get_month_statistics.
// @Summary		Estatísticas agregadas do mês
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		month_id	path	string	true	"UUID training_months"
// @Success		200	{object}	models.MonthStatisticsRow
// @Router		/me/training/months/{month_id}/statistics [get]
func (c *TrainingProgressController) GetMonthStatistics(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	monthID, err := uuid.Parse(chi.URLParam(r, "month_id"))
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "month_id inválido")
		return
	}
	if !c.requireMonthInPlan(w, r, userID, monthID) {
		return
	}
	row, err := c.progress.GetMonthStatistics(userID, monthID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "sem dados de progresso para este mês")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(row)
}

func (c *TrainingProgressController) getMonthReport(w http.ResponseWriter, r *http.Request, fn func(userID, monthID uuid.UUID) (interface{}, error)) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	monthID, err := uuid.Parse(chi.URLParam(r, "month_id"))
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "month_id inválido")
		return
	}
	if !c.requireMonthInPlan(w, r, userID, monthID) {
		return
	}
	data, err := fn(userID, monthID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(data)
}
