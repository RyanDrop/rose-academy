package controllers

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"rose-academy/api"
	"rose-academy/middleware"
	"rose-academy/models"
)

// TrainingPlansController lista planos de treino com meses (catálogo público).
type TrainingPlansController struct {
	repo         models.TrainingPlanRepository
	userTraining models.UserTrainingRepository
	devMode      bool
}

// NewTrainingPlansController cria o controller. devMode: erros 500 incluem detalhe.
func NewTrainingPlansController(repo models.TrainingPlanRepository, userTraining models.UserTrainingRepository, devMode bool) *TrainingPlansController {
	return &TrainingPlansController{repo: repo, userTraining: userTraining, devMode: devMode}
}

// List planos ativos com meses ativos (sem is_active, created_at, updated_at).
// @Summary		Listar planos de treino com meses
// @Description	Retorna training_plans (is_active) e training_months (is_active); title=name, subtitle=description; level em PT-BR com acentos (Iniciante, Intermediário, Avançado) mesmo que o banco guarde sem acentos. Cada mês inclui active: com Bearer e plano ativo, a mesma regra de tempo (selected_at do plano ativo + meses de calendário, 1..12) aplica-se a todos os níveis do catálogo (mês 1 e 2 ativos nos três planos após 1 mês de uso, etc.); sem token ou sem plano, active=false.
// @Tags			catalog
// @Produce		json
// @Success		200	{array}	models.TrainingPlanPublic
// @Router		/training-plans [get]
func (c *TrainingPlansController) List(w http.ResponseWriter, r *http.Request) {
	plans, err := c.repo.ListActivePlansWithMonths()
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if plans == nil {
		plans = []*models.TrainingPlanPublic{}
	}
	if err := c.applyMonthsActive(r, plans); err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(plans)
}

func (c *TrainingPlansController) applyMonthsActive(r *http.Request, plans []*models.TrainingPlanPublic) error {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		for _, pl := range plans {
			for _, m := range pl.Months {
				m.Active = false
			}
		}
		return nil
	}
	plan, err := c.userTraining.GetActivePlan(userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			for _, pl := range plans {
				for _, m := range pl.Months {
					m.Active = false
				}
			}
			return nil
		}
		for _, pl := range plans {
			for _, m := range pl.Months {
				m.Active = false
			}
		}
		return err
	}
	now := time.Now()
	for _, pl := range plans {
		for _, m := range pl.Months {
			m.Active = models.TrainingMonthActiveForPlan(m.MonthNumber, plan.SelectedAt, now)
		}
	}
	return nil
}
