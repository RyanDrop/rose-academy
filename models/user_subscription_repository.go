package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrSubscriptionNotFoundForBilling retornado quando não existe user_subscription com o abp_billing_id informado
var ErrSubscriptionNotFoundForBilling = errors.New("nenhuma assinatura encontrada para este billing_id")

// PostgresUserSubscriptionRepository implementação do UserSubscriptionRepository
type PostgresUserSubscriptionRepository struct {
	db *sql.DB
}

// NewPostgresUserSubscriptionRepository cria repositório de assinaturas
func NewPostgresUserSubscriptionRepository(db *sql.DB) *PostgresUserSubscriptionRepository {
	return &PostgresUserSubscriptionRepository{db: db}
}

// Create insere uma nova assinatura (cobrança criada, aguardando pagamento)
func (r *PostgresUserSubscriptionRepository) Create(s *UserSubscription) error {
	now := time.Now()
	q := `INSERT INTO user_subscriptions (user_id, product_id, abp_billing_id, status, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6) RETURNING id, created_at, updated_at`
	return r.db.QueryRow(q, s.UserID, s.ProductID, s.ABPBillingID, s.Status, now, now).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

// FindByABPBillingID retorna assinatura pelo id da cobrança na Abacate Pay
func (r *PostgresUserSubscriptionRepository) FindByABPBillingID(abpBillingID string) (*UserSubscription, error) {
	row := r.db.QueryRow(`
		SELECT id, user_id, product_id, abp_billing_id, status, paid_at, created_at, updated_at
		FROM user_subscriptions WHERE abp_billing_id = $1
	`, abpBillingID)
	var s UserSubscription
	var paidAt sql.NullTime
	err := row.Scan(&s.ID, &s.UserID, &s.ProductID, &s.ABPBillingID, &s.Status, &paidAt, &s.CreatedAt, &s.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if paidAt.Valid {
		s.PaidAt = &paidAt.Time
	}
	return &s, nil
}

// UpdateStatus atualiza status e paid_at da assinatura
func (r *PostgresUserSubscriptionRepository) UpdateStatus(abpBillingID, status string, paidAt *time.Time) error {
	now := time.Now()
	var result sql.Result
	var err error
	if paidAt != nil {
		result, err = r.db.Exec(`UPDATE user_subscriptions SET status = $1, paid_at = $2, updated_at = $3 WHERE abp_billing_id = $4`, status, paidAt, now, abpBillingID)
	} else {
		result, err = r.db.Exec(`UPDATE user_subscriptions SET status = $1, updated_at = $2 WHERE abp_billing_id = $3`, status, now, abpBillingID)
	}
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSubscriptionNotFoundForBilling
	}
	return nil
}

// FindByUserID retorna assinaturas do usuário
func (r *PostgresUserSubscriptionRepository) FindByUserID(userID uuid.UUID) ([]*UserSubscription, error) {
	rows, err := r.db.Query(`
		SELECT id, user_id, product_id, abp_billing_id, status, paid_at, created_at, updated_at
		FROM user_subscriptions WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*UserSubscription
	for rows.Next() {
		var s UserSubscription
		var paidAt sql.NullTime
		if err := rows.Scan(&s.ID, &s.UserID, &s.ProductID, &s.ABPBillingID, &s.Status, &paidAt, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, err
		}
		if paidAt.Valid {
			s.PaidAt = &paidAt.Time
		}
		list = append(list, &s)
	}
	return list, rows.Err()
}
