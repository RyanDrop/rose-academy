package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"
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

// AuthTokenResponse resposta de registro ou login (JWT + usuário)
type AuthTokenResponse struct {
	User  models.User `json:"user"`
	Token string      `json:"token"`
}

// RegisterRequest body esperado para POST /auth/register (obrigatórios: email, password, full_name; tax_id opcional).
type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	TaxID    string `json:"tax_id,omitempty"`
}

// Register cadastro de usuário e criação de cliente Abacate Pay
// @Summary		Cadastro
// @Description	Cria conta com email, senha e nome completo; opcional tax_id (CPF/CNPJ, só dígitos no banco). Cliente na Abacate Pay; retorna JWT. Celular, meta e gênero em PATCH /auth/me.
// @Tags			auth
// @Accept		json
// @Produce		json
// @Param		body	body	RegisterRequest	true	"email, password, full_name; opcional tax_id"
// @Success		201	{object}	AuthTokenResponse
// @Failure		400	{object}	map[string]interface{}	"validação"
// @Failure		409	{object}	map[string]interface{}	"email ou CPF em uso"
// @Failure		503	{object}	map[string]interface{}	"gateway de pagamento"
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
	if len(fieldErrs) > 0 {
		api.WriteValidation(w, r, fieldErrs)
		return
	}
	in := services.RegisterInput{
		Email:    req.Email,
		Password: req.Password,
		FullName: req.FullName,
		TaxID:    req.TaxID,
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

// Login autenticação por email e senha
// @Summary		Login
// @Tags			auth
// @Accept		json
// @Produce		json
// @Param		body	body	LoginRequest	true	"email e senha"
// @Success		200	{object}	AuthTokenResponse
// @Failure		400	{object}	map[string]interface{}	"validação"
// @Failure		401	{object}	map[string]interface{}	"credenciais inválidas"
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

// Me perfil do usuário autenticado
// @Summary		Perfil (me)
// @Tags			auth
// @Produce		json
// @Security		BearerAuth
// @Success		200	{object}	models.User
// @Failure		401	{object}	map[string]interface{}
// @Failure		404	{object}	map[string]interface{}
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
	DateBirth     *string  `json:"date_birth,omitempty"` // ISO date "2006-01-02"
	Weight        *float64 `json:"weight,omitempty"`
	Height        *float64 `json:"height,omitempty"`
	GoalID        *string  `json:"goal_id,omitempty"`
	ProfileImg    *string  `json:"profile_img,omitempty"` // base64
	Cellphone     *string  `json:"cellphone,omitempty"`
	TaxID         *string  `json:"tax_id,omitempty"` // CPF/CNPJ (apenas dígitos gravados); string vazia remove
	Gender        *string  `json:"gender,omitempty"` // feminino, masculino, outro, prefiro_nao_dizer; vazio remove
	EmailVerified *bool    `json:"email_verified,omitempty"`
}

// PatchMe atualização parcial do perfil
// @Summary		Atualizar perfil
// @Tags			auth
// @Accept		json
// @Produce		json
// @Security		BearerAuth
// @Param		body	body	PatchMeRequest	true	"campos opcionais"
// @Success		200	{object}	models.User
// @Failure		400	{object}	map[string]interface{}	"validação"
// @Failure		401	{object}	map[string]interface{}
// @Failure		409	{object}	map[string]interface{}	"CPF/CNPJ em uso"
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
	var taxID *string
	if req.TaxID != nil {
		norm := digitsOnly(*req.TaxID)
		if norm != "" {
			other, err := c.userRepo.FindByTaxID(norm)
			if err != nil {
				api.WriteInternalErr(w, r, err, c.devMode)
				return
			}
			if other != nil && other.ID != userID {
				api.Write(w, r, http.StatusConflict, "Conflict", "CPF/CNPJ já cadastrado")
				return
			}
			nCopy := norm
			taxID = &nCopy
		} else {
			empty := ""
			eCopy := empty
			taxID = &eCopy
		}
	}
	var gender *string
	if req.Gender != nil {
		g, ok := normalizeGenderInput(*req.Gender)
		if !ok {
			fieldErrs = append(fieldErrs, api.FieldError{Field: "gender", Message: "valor inválido; use feminino, masculino, outro ou prefiro_nao_dizer"})
		} else {
			gCopy := g
			gender = &gCopy
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
		TaxID:         taxID,
		Gender:        gender,
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

func digitsOnly(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}

// normalizeGenderInput retorna o valor canônico do enum ou "" para limpar; ok false se inválido (e não vazio).
func normalizeGenderInput(s string) (string, bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return "", true
	}
	switch s {
	case models.GenderFeminino, models.GenderMasculino, models.GenderOutro, models.GenderPrefiroNaoDizer:
		return s, true
	default:
		return "", false
	}
}
