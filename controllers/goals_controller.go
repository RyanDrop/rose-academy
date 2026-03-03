package controllers

import (
	"encoding/json"
	"net/http"

	"rose-academy/api"
	"rose-academy/models"
)

// GoalsController controller para listar metas (ex.: para formulário de cadastro)
type GoalsController struct {
	goalRepo models.GoalRepository
	devMode  bool
}

// NewGoalsController cria o controller de metas. devMode: quando true, erros 500 incluem a mensagem real.
func NewGoalsController(goalRepo models.GoalRepository, devMode bool) *GoalsController {
	return &GoalsController{goalRepo: goalRepo, devMode: devMode}
}

// @Router		/goals [get]
func (c *GoalsController) List(w http.ResponseWriter, r *http.Request) {
	goals, err := c.goalRepo.FindAll()
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(goals)
}
