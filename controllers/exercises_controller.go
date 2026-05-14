package controllers

import (
	"encoding/json"
	"net/http"
	"strings"

	"rose-academy/api"
	"rose-academy/models"
)

// ExercisesController biblioteca pública de exercícios.
type ExercisesController struct {
	repo    models.ExerciseRepository
	devMode bool
}

// NewExercisesController cria o controller.
func NewExercisesController(repo models.ExerciseRepository, devMode bool) *ExercisesController {
	return &ExercisesController{repo: repo, devMode: devMode}
}

// List exercícios ativos; query opcional `q` filtra nome, músculo alvo e equipamento (ILIKE).
// @Summary		Listar exercícios
// @Description	Biblioteca compartilhada (exercises.is_active = true).
// @Tags			training
// @Produce		json
// @Param		q	query	string	false	"termo de busca"
// @Success		200	{array}	models.ExercisePublic
// @Router		/exercises [get]
func (c *ExercisesController) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	list, err := c.repo.ListActive(q)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if list == nil {
		list = []*models.ExercisePublic{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(list)
}
