package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"rose-academy/api"
	"rose-academy/middleware"
	"rose-academy/models"
)

// TrainingWorkoutsController treinos semanais (parte 5).
type TrainingWorkoutsController struct {
	repo    models.TrainingWorkoutRepository
	devMode bool
}

// NewTrainingWorkoutsController cria o controller.
func NewTrainingWorkoutsController(repo models.TrainingWorkoutRepository, devMode bool) *TrainingWorkoutsController {
	return &TrainingWorkoutsController{repo: repo, devMode: devMode}
}

// ListMonthWorkouts lista treinos do mês (get_month_workouts); o mês deve pertencer ao plano ativo.
// @Summary		Treinos do mês
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		month_id	path	string	true	"UUID do training_months"
// @Success		200	{array}	models.MonthWorkoutSummary
// @Failure		400	{object}	map[string]interface{}
// @Failure		403	{object}	map[string]interface{}
// @Router		/me/training/months/{month_id}/workouts [get]
func (c *TrainingWorkoutsController) ListMonthWorkouts(w http.ResponseWriter, r *http.Request) {
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
	allowed, err := c.repo.IsMonthInUserActivePlan(userID, monthID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if !allowed {
		api.Write(w, r, http.StatusForbidden, "Forbidden", "este mês não pertence ao seu plano ativo")
		return
	}
	list, err := c.repo.ListMonthWorkouts(monthID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if list == nil {
		list = []*models.MonthWorkoutSummary{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(list)
}

// GetWorkoutOfDay treino do dia no mês em progresso ou em `month_id` (query).
// @Summary		Treino do dia da semana
// @Description	day_number 1–7 (weekly_workouts). Use query `month_id` para outro mês; sem query, usa o mês em progresso não concluído (user_month_progress).
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		day_number	path	int		true	"dia do treino (1-7)"
// @Param		month_id	query	string	false	"UUID do mês (opcional)"
// @Success		200	{object}	models.WorkoutDetailResponse
// @Failure		400	{object}	map[string]interface{}
// @Failure		404	{object}	map[string]interface{}
// @Router		/me/training/workouts/day/{day_number} [get]
func (c *TrainingWorkoutsController) GetWorkoutOfDay(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	dn, err := strconv.Atoi(chi.URLParam(r, "day_number"))
	if err != nil || dn < 1 || dn > 7 {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "day_number deve ser inteiro entre 1 e 7")
		return
	}
	var monthID uuid.UUID
	rawMonth := r.URL.Query().Get("month_id")
	if rawMonth != "" {
		monthID, err = uuid.Parse(rawMonth)
		if err != nil {
			api.Write(w, r, http.StatusBadRequest, "Bad Request", "month_id inválido")
			return
		}
		allowed, err := c.repo.IsMonthInUserActivePlan(userID, monthID)
		if err != nil {
			api.WriteInternalErr(w, r, err, c.devMode)
			return
		}
		if !allowed {
			api.Write(w, r, http.StatusForbidden, "Forbidden", "este mês não pertence ao seu plano ativo")
			return
		}
	} else {
		monthID, err = c.repo.ResolveInProgressMonthID(userID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				api.Write(w, r, http.StatusNotFound, "Not Found", "nenhum mês em progresso; informe month_id na query ou inicie um mês")
				return
			}
			api.WriteInternalErr(w, r, err, c.devMode)
			return
		}
	}
	detail, err := c.repo.GetWorkoutOfDayForMonth(userID, monthID, dn)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "não há treino para este dia neste mês")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(detail)
}

// GetWorkout detalhe de um treino por id (se pertencer ao plano ativo).
// @Summary		Detalhe do treino
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		workout_id	path	string	true	"UUID de weekly_workouts"
// @Success		200	{object}	models.WorkoutDetailResponse
// @Failure		400	{object}	map[string]interface{}
// @Failure		404	{object}	map[string]interface{}
// @Router		/me/training/workouts/{workout_id} [get]
func (c *TrainingWorkoutsController) GetWorkout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
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
	detail, err := c.repo.GetWorkoutDetailForUser(userID, workoutID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "treino não encontrado ou não disponível no seu plano")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(detail)
}
