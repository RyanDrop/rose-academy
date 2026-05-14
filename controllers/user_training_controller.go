package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"rose-academy/api"
	"rose-academy/middleware"
	"rose-academy/models"
)

// UserTrainingController plano e meses do usuário autenticado (parte 4).
type UserTrainingController struct {
	repo    models.UserTrainingRepository
	devMode bool
}

// NewUserTrainingController cria o controller.
func NewUserTrainingController(repo models.UserTrainingRepository, devMode bool) *UserTrainingController {
	return &UserTrainingController{repo: repo, devMode: devMode}
}

// SelectTrainingPlanRequest body POST /me/training/plan.
type SelectTrainingPlanRequest struct {
	Level string `json:"level"`
}

// SelectPlan define o nível do plano ativo (select_training_plan).
// @Summary		Escolher plano de treino
// @Description	Desativa seleções anteriores e ativa o plano do nível informado (iniciante, intermediario, avancado; aceita variações com acento).
// @Tags			training
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param		body	body	SelectTrainingPlanRequest	true	"nivel: level"
// @Success		200	{object}	models.SelectTrainingPlanResponse
// @Failure		400	{object}	map[string]interface{}
// @Failure		404	{object}	map[string]interface{}
// @Router		/me/training/plan [post]
func (c *UserTrainingController) SelectPlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	var req SelectTrainingPlanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	enum, valid := models.ToTrainingLevelDBEnum(req.Level)
	if !valid {
		api.WriteValidation(w, r, []api.FieldError{
			{Field: "level", Message: "use iniciante, intermediario ou avancado"},
		})
		return
	}
	selectionID, err := c.repo.SelectTrainingPlan(userID, enum)
	if err != nil {
		if models.IsPGRaiseException(err) || strings.Contains(err.Error(), "não encontrado") {
			api.Write(w, r, http.StatusNotFound, "Not Found", "plano não encontrado para o nível informado")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(models.SelectTrainingPlanResponse{SelectionID: selectionID})
}

// GetActivePlan retorna o plano ativo (get_user_active_plan).
// @Summary		Plano de treino ativo
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	models.UserActivePlanResponse
// @Failure		404	{object}	map[string]interface{}
// @Router		/me/training/plan [get]
func (c *UserTrainingController) GetActivePlan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	plan, err := c.repo.GetActivePlan(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "nenhum plano de treino ativo")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(plan)
}

// ListPlanMonths lista os 12 meses com status (get_user_plan_months).
// @Summary		Meses do plano ativo
// @Description	Requer plano ativo; caso contrário 404. O campo active: sem nenhum mês de treino iniciado (user_month_progress), só o mês 1 fica true; após iniciar algum mês, segue selected_at e meses de calendário (1..12).
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Success		200	{array}	models.UserTrainingMonthItem
// @Failure		404	{object}	map[string]interface{}
// @Router		/me/training/months [get]
func (c *UserTrainingController) ListPlanMonths(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	months, err := c.repo.ListPlanMonths(userID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if len(months) == 0 {
		active, err := c.repo.HasActiveTrainingSelection(userID)
		if err != nil {
			api.WriteInternalErr(w, r, err, c.devMode)
			return
		}
		if !active {
			api.Write(w, r, http.StatusNotFound, "Not Found", "nenhum plano de treino ativo")
			return
		}
	}
	plan, err := c.repo.GetActivePlan(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "nenhum plano de treino ativo")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	hasStarted, err := c.repo.HasUserStartedAnyTrainingMonth(userID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	now := time.Now()
	for _, m := range months {
		m.Active = models.TrainingMonthActiveForList(m.MonthNumber, plan.SelectedAt, now, hasStarted)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(months)
}

// StartMonth inicia ou retoma progresso no mês (start_month_progress).
// @Summary		Iniciar mês do plano
// @Description	month_number de 1 a 12. Sem nenhum mês de treino iniciado, só é permitido iniciar o mês 1; depois aplica-se o mesmo critério de active que GET /me/training/months (selected_at + calendário).
// @Tags			training
// @Produce		json
// @Security		BearerAuth
// @Param		month_number	path	int	true	"número do mês (1-12)"
// @Success		200	{object}	models.StartMonthProgressResponse
// @Failure		400	{object}	map[string]interface{}
// @Failure		401	{object}	map[string]interface{}
// @Failure		403	{object}	map[string]interface{}
// @Failure		404	{object}	map[string]interface{}
// @Router		/me/training/months/{month_number}/start [post]
func (c *UserTrainingController) StartMonth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	raw := chi.URLParam(r, "month_number")
	n, err := strconv.Atoi(raw)
	if err != nil || n < 1 || n > 12 {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "month_number deve ser um inteiro entre 1 e 12")
		return
	}
	plan, err := c.repo.GetActivePlan(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "nenhum plano de treino ativo")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	hasStarted, err := c.repo.HasUserStartedAnyTrainingMonth(userID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if !hasStarted && n != 1 {
		api.Write(w, r, http.StatusForbidden, "Forbidden", "inicie o mês 1 do plano antes de iniciar outros meses")
		return
	}
	if hasStarted && !models.TrainingMonthActiveForPlan(n, plan.SelectedAt, time.Now()) {
		api.Write(w, r, http.StatusForbidden, "Forbidden", "este mês ainda não está liberado pelo cronograma do plano atual")
		return
	}
	progressID, err := c.repo.StartMonthProgress(userID, n)
	if err != nil {
		if models.IsPGRaiseException(err) || strings.Contains(strings.ToLower(err.Error()), "não encontrado") {
			api.Write(w, r, http.StatusNotFound, "Not Found", "mês não encontrado para o plano ativo ou plano inativo")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(models.StartMonthProgressResponse{ProgressID: progressID})
}
