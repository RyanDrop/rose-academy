package controllers

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/google/uuid"

	"rose-academy/api"
	"rose-academy/middleware"
	"rose-academy/models"
	"rose-academy/services"
)

// AuthController controller de autenticação (registro e login)
type AuthController struct {
	authService *services.AuthService
	userRepo    models.UserRepository
	jwtConfig   middleware.JWTConfig
	devMode     bool // em true, erros 500 incluem a mensagem real (para depuração)
}

// NewAuthController cria o controller de autenticação. devMode: quando true (ex.: ENV=development), o detail do 500 usa a mensagem do erro.
func NewAuthController(authService *services.AuthService, userRepo models.UserRepository, jwtConfig middleware.JWTConfig, devMode bool) *AuthController {
	return &AuthController{
		authService: authService,
		userRepo:    userRepo,
		jwtConfig:   jwtConfig,
		devMode:     devMode,
	}
}

func (c *AuthController) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// RegisterRequest body esperado para POST /auth/register
type RegisterRequest struct {
	Email     string  `json:"email"`
	Password  string  `json:"password"`
	FullName  string  `json:"full_name"`
	Cellphone string  `json:"cellphone"`
	TaxID     string  `json:"tax_id"`
	GoalID    *string `json:"goal_id,omitempty"`
	Gender    *string `json:"gender,omitempty"`
}

// Register cria usuário no app e cliente na Abacate Pay; retorna usuário + token
// @Summary		Criar usuário e cliente Abacate Pay
// @Description	Cadastra usuário no app e cria cliente na Abacate Pay. Retorna usuário e JWT.
// @Tags			auth
// @Accept		json
// @Produce		json
// @Param		body	body		RegisterRequest	true	"Dados do cadastro"
// @Success		201	{object}	map[string]interface{}
// @Failure		400	{object}	map[string]string
// @Failure		409	{object}	map[string]string	"email já cadastrado"
// @Router		/auth/register [post]
func (c *AuthController) Register(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	var req RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	var fieldErrs []api.FieldError
	if req.Email == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "email", Message: "é obrigatório"})
	}
	if req.Password == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "password", Message: "é obrigatório"})
	}
	if req.FullName == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "full_name", Message: "é obrigatório"})
	}
	if req.Cellphone == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "cellphone", Message: "é obrigatório"})
	}
	if req.TaxID == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "tax_id", Message: "é obrigatório"})
	}
	if len(fieldErrs) > 0 {
		api.WriteValidation(w, r, fieldErrs)
		return
	}
	var goalID *uuid.UUID
	if req.GoalID != nil && *req.GoalID != "" {
		id, err := uuid.Parse(*req.GoalID)
		if err != nil {
			api.Write(w, r, http.StatusBadRequest, "Bad Request", "goal_id inválido")
			return
		}
		goalID = &id
	}
	in := services.RegisterInput{
		Email:     req.Email,
		Password:  req.Password,
		FullName:  req.FullName,
		Cellphone: req.Cellphone,
		TaxID:     req.TaxID,
		GoalID:    goalID,
		Gender:    req.Gender,
	}
	user, err := c.authService.Register(in)
	if err != nil {
		if errors.Is(err, services.ErrEmailInUse) {
			api.Write(w, r, http.StatusConflict, "Conflict", "email já cadastrado")
			return
		}
		if errors.Is(err, services.ErrTaxIDInUse) {
			api.Write(w, r, http.StatusConflict, "Conflict", "CPF/CNPJ já cadastrado")
			return
		}
		if errors.Is(err, services.ErrPaymentGateway) {
			api.Write(w, r, http.StatusServiceUnavailable, "Service Unavailable", "Gateway de pagamento não configurado. Configure ABACATE_PAY_API_KEY.")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	token, err := c.jwtConfig.GenerateToken(user.ID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	c.respondJSON(w, http.StatusCreated, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// LoginRequest body esperado para POST /auth/login
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login valida credenciais e retorna usuário + token
// @Summary		Login
// @Description	Valida email e senha; retorna usuário e JWT.
// @Tags			auth
// @Accept		json
// @Produce		json
// @Param		body	body		LoginRequest	true	"Credenciais"
// @Success		200	{object}	map[string]interface{}
// @Failure		400	{object}	map[string]string
// @Failure		401	{object}	map[string]string
// @Router		/auth/login [post]
func (c *AuthController) Login(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	var fieldErrs []api.FieldError
	if req.Email == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "email", Message: "é obrigatório"})
	}
	if req.Password == "" {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "password", Message: "é obrigatório"})
	}
	if len(fieldErrs) > 0 {
		api.WriteValidation(w, r, fieldErrs)
		return
	}
	user, err := c.authService.Login(services.LoginInput{Email: req.Email, Password: req.Password})
	if err != nil {
		if errors.Is(err, services.ErrInvalidCreds) {
			api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "email ou senha inválidos")
			return
		}
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	token, err := c.jwtConfig.GenerateToken(user.ID)
	if err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
		return
	}
	c.respondJSON(w, http.StatusOK, map[string]interface{}{
		"user":  user,
		"token": token,
	})
}

// Me retorna o usuário autenticado (requer Authorization: Bearer <token>)
// @Summary		Usuário logado
// @Description	Retorna os dados do usuário autenticado. Requer header Authorization: Bearer &lt;token&gt;.
// @Tags			auth
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	models.User
// @Failure		401	{object}	map[string]string
// @Failure		404	{object}	map[string]string
// @Router		/auth/me [get]
func (c *AuthController) Me(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	user, err := c.userRepo.FindByID(userID)
	if err != nil || user == nil {
		api.Write(w, r, http.StatusNotFound, "Not Found", "usuário não encontrado")
		return
	}
	user.PasswordHash = ""
	c.respondJSON(w, http.StatusOK, user)
}
