package controllers

import (
	"encoding/json"
	"net/http"

	"rose-academy/api"
	"rose-academy/models"
)

// TrainingPlansController lista planos de treino com meses (catálogo público).
type TrainingPlansController struct {
	repo    models.TrainingPlanRepository
	devMode bool
}

// NewTrainingPlansController cria o controller. devMode: erros 500 incluem detalhe.
func NewTrainingPlansController(repo models.TrainingPlanRepository, devMode bool) *TrainingPlansController {
	return &TrainingPlansController{repo: repo, devMode: devMode}
}

// List planos ativos com meses ativos (sem is_active, created_at, updated_at).
// @Summary		Listar planos de treino com meses
// @Description	Retorna training_plans (is_active) e training_months (is_active); title=name, subtitle=description; level em PT-BR com acentos (Iniciante, Intermediário, Avançado) mesmo que o banco guarde sem acentos.
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
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(plans)
}
