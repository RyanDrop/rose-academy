package services

import (
	"database/sql"
	"errors"
	"time"

	"golang.org/x/crypto/bcrypt"

	"rose-academy/models"
)

var (
	ErrEmailInUse     = errors.New("email já cadastrado")
	ErrTaxIDInUse     = errors.New("CPF/CNPJ já cadastrado")
	ErrInvalidCreds   = errors.New("email ou senha inválidos")
	ErrPaymentGateway = errors.New("gateway de pagamento indisponível ou não configurado")
)

// AuthService serviço de autenticação e registro
type AuthService struct {
	db          *sql.DB
	userRepo    models.UserRepository
	goalRepo    models.GoalRepository
	abpRepo     models.ABPCustomerRepository
	abpClient   *AbacatePayClient
	bcryptCost  int
}

// NewAuthService cria o serviço de autenticação
func NewAuthService(
	db *sql.DB,
	userRepo models.UserRepository,
	goalRepo models.GoalRepository,
	abpRepo models.ABPCustomerRepository,
	abpClient *AbacatePayClient,
) *AuthService {
	return &AuthService{
		db:          db,
		userRepo:    userRepo,
		goalRepo:    goalRepo,
		abpRepo:     abpRepo,
		abpClient:   abpClient,
		bcryptCost:  bcrypt.DefaultCost,
	}
}

// RegisterInput dados para registro (CPF opcional; demais perfil em PATCH /auth/me).
type RegisterInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	FullName string `json:"full_name"`
	TaxID    string `json:"tax_id,omitempty"` // apenas dígitos gravados; vazio ignora
}

// Register cria o cliente na Abacate Pay e só então persiste usuário e vínculo no banco em uma transação.
// Se a criação na Abacate Pay falhar, nada é gravado no banco.
func (s *AuthService) Register(in RegisterInput) (*models.User, error) {
	existing, _ := s.userRepo.FindByEmail(in.Email)
	if existing != nil {
		return nil, ErrEmailInUse
	}
	taxNorm := onlyDigits(in.TaxID)
	if taxNorm != "" {
		byTax, _ := s.userRepo.FindByTaxID(taxNorm)
		if byTax != nil {
			return nil, ErrTaxIDInUse
		}
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(in.Password), s.bcryptCost)
	if err != nil {
		return nil, err
	}
	user := &models.User{
		Email:         in.Email,
		PasswordHash:  string(hash),
		FullName:      in.FullName,
		IsActive:      true,
		EmailVerified: false,
	}
	if taxNorm != "" {
		user.TaxID = taxNorm
	}

	// 1) Criar cliente na Abacate Pay antes de qualquer write no banco.
	abpID, err := s.abpClient.CreateCustomer(user.FullName, "", user.Email, user.TaxID)
	if err != nil {
		if errors.Is(err, ErrAPIKeyNotConfigured) {
			return nil, ErrPaymentGateway
		}
		return nil, err
	}

	// 2) Transação: usuário + abp_customers; se qualquer passo falhar, nada é commitado
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if err := s.userRepo.CreateInTx(tx, user); err != nil {
		return nil, err
	}
	abpCustomer := &models.ABPCustomer{
		UserID:        user.ID,
		ABPCustomerID: abpID,
		ABPEmail:      &user.Email,
		LastSyncAt:    time.Now(),
	}
	if err := s.abpRepo.CreateInTx(tx, abpCustomer); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}

	user.PasswordHash = ""
	return user, nil
}

func onlyDigits(s string) string {
	var b []byte
	for i := 0; i < len(s); i++ {
		if s[i] >= '0' && s[i] <= '9' {
			b = append(b, s[i])
		}
	}
	return string(b)
}

// LoginInput dados para login
type LoginInput struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// Login valida credenciais e atualiza last_login_at; retorna o usuário (sem senha) e erro
func (s *AuthService) Login(in LoginInput) (*models.User, error) {
	user, err := s.userRepo.FindByEmail(in.Email)
	if err != nil || user == nil {
		return nil, ErrInvalidCreds
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(in.Password)); err != nil {
		return nil, ErrInvalidCreds
	}
	_ = s.userRepo.UpdateLastLoginAt(user.ID, time.Now())
	user.PasswordHash = ""
	user.LastLoginAt = func(t time.Time) *time.Time { return &t }(time.Now())
	return user, nil
}
