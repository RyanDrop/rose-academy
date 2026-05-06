package controllers

import (
	"encoding/json"
	"net/http"

	"rose-academy/api"
	"rose-academy/models"
)

// ProductsController controller para listar produtos/planos (Abacate Pay)
type ProductsController struct {
	productRepo models.ProductRepository
	devMode     bool
}

// NewProductsController cria o controller de produtos
func NewProductsController(productRepo models.ProductRepository, devMode bool) *ProductsController {
	return &ProductsController{productRepo: productRepo, devMode: devMode}
}

// List lista produtos/planos (requer JWT)
// @Summary		Listar produtos
// @Tags			catalog
// @Produce		json
// @Security		BearerAuth
// @Success		200	{array}	models.Product
// @Failure		401	{object}	map[string]interface{}
// @Router		/products [get]
func (c *ProductsController) List(w http.ResponseWriter, r *http.Request) {
	products, err := c.productRepo.FindAll()
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	if products == nil {
		products = []*models.Product{}
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_ = json.NewEncoder(w).Encode(products)
}
