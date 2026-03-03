package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"

	"rose-academy/api"
	"rose-academy/middleware"
	"rose-academy/models"
	"rose-academy/services"
)

type AuthController struct {
	authService *services.AuthService
	userRepo    models.UserRepository
	goalRepo    models.GoalRepository
	jwtConfig   middleware.JWTConfig
	devMode     bool // em true, erros 500 incluem a mensagem real (para depuração)
}

// NewAuthController cria o controller de autenticação. devMode: quando true (ex.: ENV=development), o detail do 500 usa a mensagem do erro.
func NewAuthController(authService *services.AuthService, userRepo models.UserRepository, goalRepo models.GoalRepository, jwtConfig middleware.JWTConfig, devMode bool) *AuthController {
	return &AuthController{
		authService: authService,
		userRepo:    userRepo,
		goalRepo:    goalRepo,
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

// PatchMeRequest body para PATCH /auth/me (todos os campos opcionais)
type PatchMeRequest struct {
	DateBirth     *string  `json:"date_birth,omitempty"`     // ISO date "2006-01-02"
	Weight        *float64 `json:"weight,omitempty"`
	Height        *float64 `json:"height,omitempty"`
	GoalID        *string  `json:"goal_id,omitempty"`
	ProfileImg    *string  `json:"profile_img,omitempty"`    // base64
	Cellphone     *string  `json:"cellphone,omitempty"`
	EmailVerified *bool    `json:"email_verified,omitempty"`
}

// @Router		/auth/me [patch]
func (c *AuthController) PatchMe(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPatch {
		api.Write(w, r, http.StatusMethodNotAllowed, "Method Not Allowed", "método não permitido")
		return
	}
	userID, ok := middleware.GetUserID(r.Context())
	if !ok {
		api.Write(w, r, http.StatusUnauthorized, "Unauthorized", "não autenticado")
		return
	}
	var req PatchMeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.Write(w, r, http.StatusBadRequest, "Bad Request", "body JSON inválido")
		return
	}
	var fieldErrs []api.FieldError
	var dateBirth *time.Time
	if req.DateBirth != nil {
		t, err := time.Parse("2006-01-02", *req.DateBirth)
		if err != nil {
			fieldErrs = append(fieldErrs, api.FieldError{Field: "date_birth", Message: "deve ser uma data no formato AAAA-MM-DD"})
		} else {
			dateBirth = &t
		}
	}
	if req.Weight != nil && *req.Weight <= 0 {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "weight", Message: "deve ser maior que zero"})
	}
	if req.Height != nil && *req.Height <= 0 {
		fieldErrs = append(fieldErrs, api.FieldError{Field: "height", Message: "deve ser maior que zero"})
	}
	var goalID *uuid.UUID
	if req.GoalID != nil && *req.GoalID != "" {
		id, err := uuid.Parse(*req.GoalID)
		if err != nil {
			fieldErrs = append(fieldErrs, api.FieldError{Field: "goal_id", Message: "UUID inválido"})
		} else {
			goal, err := c.goalRepo.FindByID(id)
			if err != nil || goal == nil {
				fieldErrs = append(fieldErrs, api.FieldError{Field: "goal_id", Message: "meta não encontrada"})
			} else {
				goalID = &id
			}
		}
	}
	if len(fieldErrs) > 0 {
		api.WriteValidation(w, r, fieldErrs)
		return
	}
	params := models.UpdateProfileParams{
		DateBirth:     dateBirth,
		Weight:        req.Weight,
		Height:        req.Height,
		GoalID:        goalID,
		ProfileImg:    req.ProfileImg,
		Cellphone:     req.Cellphone,
		EmailVerified: req.EmailVerified,
	}
	if err := c.userRepo.UpdateProfile(userID, params); err != nil {
		api.WriteInternalErr(w, r, err, c.devMode)
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
