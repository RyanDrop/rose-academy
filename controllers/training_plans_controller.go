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
// @Description	Retorna training_plans (is_active) e training_months (is_active); title=name, subtitle=description; level em PT-BR com acentos (Iniciante, Intermediário, Avançado) mesmo que o banco guarde sem acentos. Cada mês inclui active: com Bearer e plano ativo, sem nenhum mês de treino iniciado só o mês 1 fica true em todos os níveis; após iniciar algum mês, a mesma regra de calendário (selected_at, 1..12) em todos os planos; sem token ou sem plano, active=false.
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
	hasStarted, err := c.userTraining.HasUserStartedAnyTrainingMonth(userID)
	if err != nil {
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
			m.Active = models.TrainingMonthActiveForList(m.MonthNumber, plan.SelectedAt, now, hasStarted)
		}
	}
	return nil
}
