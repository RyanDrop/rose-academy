package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"rose-academy/api"
	"rose-academy/middleware"
	"rose-academy/services"
)

// AbacateSubscriptionsController checkout de assinatura Abacate Pay API v2
type AbacateSubscriptionsController struct {
	billingService *services.BillingService
	devMode        bool
}

// NewAbacateSubscriptionsController cria o controller
func NewAbacateSubscriptionsController(billingService *services.BillingService, devMode bool) *AbacateSubscriptionsController {
	return &AbacateSubscriptionsController{billingService: billingService, devMode: devMode}
}

// SubscriptionActiveResponse resposta GET /subscriptions/active
type SubscriptionActiveResponse struct {
	// Status PAID libera acesso; PENDING o front deve listar assinaturas e ir ao checkout; null sem assinatura relevante (local).
	Status *string `json:"status"`
}

// SubscriptionActive retorna status agregado para o gate do app (PAID / PENDING / null).
// @Summary		Status da assinatura (local)
// @Description	Prioridade: PAID > PENDING com base em user_subscriptions (user_id). null se não houver PAID nem PENDING. PENDING: front lista GET /abacate/subscriptions (local) e redireciona ao checkout.
// @Tags			subscriptions
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	SubscriptionActiveResponse
// @Router		/subscriptions/active [get]
func (c *AbacateSubscriptionsController) SubscriptionActive(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	status, err := c.billingService.SubscriptionAccessStatus(userID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(SubscriptionActiveResponse{Status: status})
}

// CreateAbacateSubscriptionCheckoutRequest body POST /abacate/subscriptions/checkout
type CreateAbacateSubscriptionCheckoutRequest struct {
	ProductID     string   `json:"product_id"`
	AbpProductID  string   `json:"abp_product_id,omitempty"`
	ReturnURL     string   `json:"return_url"`
	CompletionURL string   `json:"completion_url"`
	// Assinatura v2 na Abacate aceita só CARD (PIX em conjunto gera erro na API).
	Methods []string `json:"methods,omitempty"`
}

// CreateCheckout cria checkout de assinatura (API v2). Grava user_subscriptions como PENDING e repassa a Abacate sem externalId.
// @Summary		Checkout assinatura Abacate Pay
// @Description	Cria checkout (POST /v2/subscriptions/create). Grava assinatura local PENDING; webhook marca PAID. Produto com ciclo na loja.
// @Tags			abacate-subscriptions
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param		body	body	CreateAbacateSubscriptionCheckoutRequest	true	"product_id, return_url, completion_url"
// @Success		200	{object}	services.CreateAbacateSubscriptionCheckoutOutput
// @Router		/abacate/subscriptions/checkout [post]
func (c *AbacateSubscriptionsController) CreateCheckout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	var req CreateAbacateSubscriptionCheckoutRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	pid, err := uuid.Parse(req.ProductID)
	if err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "product_id inválido")
		return
	}
	if strings.TrimSpace(req.ReturnURL) == "" || strings.TrimSpace(req.CompletionURL) == "" {
		api.WriteValidation(w, r, []api.FieldError{
			{Field: "return_url", Message: "é obrigatório"},
			{Field: "completion_url", Message: "é obrigatório"},
		})
		return
	}
	out, err := c.billingService.CreateAbacateSubscriptionCheckout(services.CreateAbacateSubscriptionCheckoutInput{
		UserID:        userID,
		ProductID:     pid,
		AbpProductID:  strings.TrimSpace(req.AbpProductID),
		ReturnURL:     strings.TrimSpace(req.ReturnURL),
		CompletionURL: strings.TrimSpace(req.CompletionURL),
		Methods:       req.Methods,
	})
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "usuário não encontrado")
			return
		}
		if errors.Is(err, services.ErrCustomerNotLinked) {
			api.Write(w, r, http.StatusBadRequest, "Bad Request", "usuário não vinculado ao Abacate Pay")
			return
		}
		if errors.Is(err, services.ErrProductNotFound) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "produto não encontrado")
			return
		}
		if errors.Is(err, services.ErrProductInactive) {
			api.Write(w, r, http.StatusBadRequest, "Bad Request", "produto inativo")
			return
		}
		if errors.Is(err, services.ErrProductMissingAbacateSubscriptionID) {
			api.Write(w, r, http.StatusBadRequest, "Bad Request", err.Error())
			return
		}
		if errors.Is(err, services.ErrAPIKeyNotConfigured) {
			api.Write(w, r, http.StatusServiceUnavailable, "Service Unavailable", "Gateway de pagamento não configurado")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(out)
}

// List lista assinaturas do usuário em user_subscriptions (sem chamar API Abacate). No máximo uma linha PAID na resposta (a mais recente).
// @Summary		Listar assinaturas (local)
// @Tags			abacate-subscriptions
// @Produce		json
// @Security		BearerAuth
// @Param		limit	query		int	false	"máximo de itens após filtros"
// @Param		id	query		string	false	"filtra por subs_*, bill_*, UUID local ou checkout_external_id"
// @Param		externalId	query		string	false	"checkout_external_id"
// @Param		status	query		string	false	"PENDING, PAID, ..."
// @Success		200	{object}	map[string]interface{}
// @Router		/abacate/subscriptions [get]
func (c *AbacateSubscriptionsController) List(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	q := r.URL.Query()
	p := services.ListSubscriptionsParams{}
	if v := q.Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Limit = n
		}
	}
	p.ID = q.Get("id")
	p.ExternalID = q.Get("externalId")
	p.Status = q.Get("status")

	raw, err := c.billingService.ListAbacateSubscriptions(userID, p)
	if err != nil {
		if errors.Is(err, services.ErrUserNotFound) {
			api.Write(w, r, http.StatusNotFound, "Not Found", "usuário não encontrado")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}

// CancelSubscriptionRequest body POST /abacate/subscriptions/cancel
type CancelSubscriptionRequest struct {
	ID string `json:"id"`
}

// Cancel cancela assinatura ativa na Abacate Pay (id retornado pela API).
// @Summary		Cancelar assinatura Abacate Pay
// @Tags			abacate-subscriptions
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param		body	body	CancelSubscriptionRequest	true	"id da assinatura (subs_...)"
// @Success		200	{object}	map[string]interface{}
// @Router		/abacate/subscriptions/cancel [post]
func (c *AbacateSubscriptionsController) Cancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	_, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	var req CancelSubscriptionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	if strings.TrimSpace(req.ID) == "" {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "id é obrigatório")
		return
	}
	raw, err := c.billingService.CancelAbacateSubscription(strings.TrimSpace(req.ID))
	if err != nil {
		if errors.Is(err, services.ErrAPIKeyNotConfigured) {
			api.Write(w, r, http.StatusServiceUnavailable, "Service Unavailable", "Gateway não configurado")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(raw)
}
