package models

import (
	"database/sql"
	"errors"
	"time"

	"github.com/google/uuid"
)

// ErrSubscriptionNotFoundForBilling retornado quando não existe user_subscription com o abp_billing_id informado
var ErrSubscriptionNotFoundForBilling = errors.New("nenhuma assinatura encontrada para este billing_id")

// ErrSubscriptionNotFoundForSubscription quando não existe linha com abp_subscription_id
var ErrSubscriptionNotFoundForSubscription = errors.New("nenhuma assinatura encontrada para este subscription_id")

// ErrWebhookNoMatchingSubscription nenhuma linha atualizada para os ids do webhook
var ErrWebhookNoMatchingSubscription = errors.New("nenhuma assinatura local corresponde ao webhook")

// PostgresUserSubscriptionRepository implementação do UserSubscriptionRepository
type PostgresUserSubscriptionRepository struct {
	db *sql.DB
}

// NewPostgresUserSubscriptionRepository cria repositório de assinaturas
func NewPostgresUserSubscriptionRepository(db *sql.DB) *PostgresUserSubscriptionRepository {
	return &PostgresUserSubscriptionRepository{db: db}
}

func scanUserSubscription(row interface{ Scan(dest ...interface{}) error }) (*UserSubscription, error) {
	var s UserSubscription
	var paidAt sql.NullTime
	var subID, extID sql.NullString
	err := row.Scan(&s.ID, &s.UserID, &s.ProductID, &s.ABPBillingID, &subID, &extID, &s.Status, &paidAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return nil, err
	}
	if subID.Valid {
		s.ABPSubscriptionID = subID.String
	}
	if extID.Valid {
		s.CheckoutExternalID = extID.String
	}
	if paidAt.Valid {
		s.PaidAt = &paidAt.Time
	}
	return &s, nil
}

const userSubSelect = `id, user_id, product_id, abp_billing_id, COALESCE(abp_subscription_id,''), COALESCE(checkout_external_id,''), status, paid_at, created_at, updated_at`

// Create insere uma nova assinatura (cobrança criada, aguardando pagamento).
// Se a Abacate reutilizar o mesmo bill_* (ex.: mesmo external_id), faz upsert em UNIQUE(user_id, abp_billing_id).
func (r *PostgresUserSubscriptionRepository) Create(s *UserSubscription) error {
	now := time.Now()
	var sub interface{}
	if s.ABPSubscriptionID != "" {
		sub = s.ABPSubscriptionID
	}
	var ext interface{}
	if s.CheckoutExternalID != "" {
		ext = s.CheckoutExternalID
	}
	q := `INSERT INTO user_subscriptions (user_id, product_id, abp_billing_id, abp_subscription_id, checkout_external_id, status, created_at, updated_at)
	      VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	      ON CONFLICT (user_id, abp_billing_id) DO UPDATE SET
	        product_id = EXCLUDED.product_id,
	        checkout_external_id = EXCLUDED.checkout_external_id,
	        abp_subscription_id = COALESCE(EXCLUDED.abp_subscription_id, user_subscriptions.abp_subscription_id),
	        status = CASE
	          WHEN user_subscriptions.status IN ('PAID', 'REFUNDED', 'CANCELLED') THEN user_subscriptions.status
	          ELSE EXCLUDED.status
	        END,
	        updated_at = now()
	      RETURNING id, created_at, updated_at`
	return r.db.QueryRow(q, s.UserID, s.ProductID, s.ABPBillingID, sub, ext, s.Status, now, now).Scan(&s.ID, &s.CreatedAt, &s.UpdatedAt)
}

// FindByABPBillingID retorna assinatura pelo id da cobrança na Abacate Pay
func (r *PostgresUserSubscriptionRepository) FindByABPBillingID(abpBillingID string) (*UserSubscription, error) {
	row := r.db.QueryRow(`
		SELECT `+userSubSelect+`
		FROM user_subscriptions WHERE abp_billing_id = $1
	`, abpBillingID)
	s, err := scanUserSubscription(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

// FindByABPSubscriptionID retorna assinatura pelo id subs_* na Abacate Pay
func (r *PostgresUserSubscriptionRepository) FindByABPSubscriptionID(subscriptionID string) (*UserSubscription, error) {
	row := r.db.QueryRow(`
		SELECT `+userSubSelect+`
		FROM user_subscriptions WHERE abp_subscription_id = $1
	`, subscriptionID)
	s, err := scanUserSubscription(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return s, nil
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

// UpdateStatusBySubscriptionID atualiza por abp_subscription_id (subs_*)
func (r *PostgresUserSubscriptionRepository) UpdateStatusBySubscriptionID(subscriptionID, status string, paidAt *time.Time) error {
	now := time.Now()
	var result sql.Result
	var err error
	if paidAt != nil {
		result, err = r.db.Exec(`UPDATE user_subscriptions SET status = $1, paid_at = $2, updated_at = $3 WHERE abp_subscription_id = $4`, status, paidAt, now, subscriptionID)
	} else {
		result, err = r.db.Exec(`UPDATE user_subscriptions SET status = $1, updated_at = $2 WHERE abp_subscription_id = $3`, status, now, subscriptionID)
	}
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSubscriptionNotFoundForSubscription
	}
	return nil
}

// SetSubscriptionIDForBilling grava subs_* quando o webhook envia o id da assinatura (ex.: após pagamento)
func (r *PostgresUserSubscriptionRepository) SetSubscriptionIDForBilling(billingID, subscriptionID string) error {
	if subscriptionID == "" {
		return nil
	}
	res, err := r.db.Exec(`UPDATE user_subscriptions SET abp_subscription_id = $1, updated_at = now() WHERE abp_billing_id = $2`, subscriptionID, billingID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return ErrSubscriptionNotFoundForBilling
	}
	return nil
}

// MarkPaidFromWebhook marca PAID tentando bill_*, subs_* ou checkout externalId (igual ao enviado no create).
func (r *PostgresUserSubscriptionRepository) MarkPaidFromWebhook(billID, subID, checkoutExt string, paidAt time.Time) error {
	now := time.Now()
	if billID != "" {
		res, err := r.db.Exec(`
			UPDATE user_subscriptions SET status = $1, paid_at = $2, updated_at = $3,
			abp_subscription_id = CASE WHEN $4 <> '' THEN $4 ELSE abp_subscription_id END
			WHERE abp_billing_id = $5`,
			SubscriptionStatusPaid, paidAt, now, subID, billID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
	}
	if subID != "" {
		res, err := r.db.Exec(`
			UPDATE user_subscriptions SET status = $1, paid_at = $2, updated_at = $3
			WHERE abp_subscription_id = $4`,
			SubscriptionStatusPaid, paidAt, now, subID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
	}
	if checkoutExt != "" {
		res, err := r.db.Exec(`
			UPDATE user_subscriptions SET status = $1, paid_at = $2, updated_at = $3,
			abp_subscription_id = CASE WHEN $4 <> '' THEN $4 ELSE abp_subscription_id END
			WHERE checkout_external_id = $5`,
			SubscriptionStatusPaid, paidAt, now, subID, checkoutExt)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
	}
	return ErrSubscriptionNotFoundForBilling
}

// UpdateStatusFromWebhook atualiza status (cancelado, reembolso, etc.) por bill, subs ou external id.
func (r *PostgresUserSubscriptionRepository) UpdateStatusFromWebhook(billID, subID, checkoutExt, status string, paidAt *time.Time) error {
	now := time.Now()
	try := func(col, val string) (bool, error) {
		var res sql.Result
		var err error
		if paidAt != nil {
			res, err = r.db.Exec(
				`UPDATE user_subscriptions SET status = $1, paid_at = $2, updated_at = $3 WHERE `+col+` = $4`,
				status, *paidAt, now, val)
		} else {
			res, err = r.db.Exec(
				`UPDATE user_subscriptions SET status = $1, updated_at = $2 WHERE `+col+` = $3`,
				status, now, val)
		}
		if err != nil {
			return false, err
		}
		n, err := res.RowsAffected()
		if err != nil {
			return false, err
		}
		return n > 0, nil
	}
	if billID != "" {
		if ok, err := try("abp_billing_id", billID); err != nil {
			return err
		} else if ok {
			return nil
		}
	}
	if subID != "" {
		if ok, err := try("abp_subscription_id", subID); err != nil {
			return err
		} else if ok {
			return nil
		}
	}
	if checkoutExt != "" {
		if ok, err := try("checkout_external_id", checkoutExt); err != nil {
			return err
		} else if ok {
			return nil
		}
	}
	return ErrSubscriptionNotFoundForBilling
}

// TouchRenewalPaidAt atualiza paid_at em renovação (mantém PAID)
func (r *PostgresUserSubscriptionRepository) TouchRenewalPaidAt(billingID, subscriptionID, checkoutExt string, paidAt time.Time) error {
	now := time.Now()
	if subscriptionID != "" {
		res, err := r.db.Exec(`
			UPDATE user_subscriptions SET paid_at = $1, status = $2, updated_at = $3
			WHERE abp_subscription_id = $4`, paidAt, SubscriptionStatusPaid, now, subscriptionID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
	}
	if billingID != "" {
		res, err := r.db.Exec(`
			UPDATE user_subscriptions SET paid_at = $1, status = $2, updated_at = $3
			WHERE abp_billing_id = $4`, paidAt, SubscriptionStatusPaid, now, billingID)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
	}
	if checkoutExt != "" {
		res, err := r.db.Exec(`
			UPDATE user_subscriptions SET paid_at = $1, status = $2, updated_at = $3
			WHERE checkout_external_id = $4`, paidAt, SubscriptionStatusPaid, now, checkoutExt)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			return nil
		}
	}
	return ErrWebhookNoMatchingSubscription
}

// FindByUserID retorna assinaturas do usuário
func (r *PostgresUserSubscriptionRepository) FindByUserID(userID uuid.UUID) ([]*UserSubscription, error) {
	rows, err := r.db.Query(`
		SELECT `+userSubSelect+`
		FROM user_subscriptions WHERE user_id = $1 ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []*UserSubscription
	for rows.Next() {
		s, err := scanUserSubscription(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, s)
	}
	return list, rows.Err()
}

// SubscriptionAccessStatus usa user_subscriptions (user_id): PAID se existir linha PAID; senão PENDING; senão null. Independente da API Abacate.
func (r *PostgresUserSubscriptionRepository) SubscriptionAccessStatus(userID uuid.UUID) (*string, error) {
	subs, err := r.FindByUserID(userID)
	if err != nil {
		return nil, err
	}
	var hasPaid, hasPending bool
	for _, s := range subs {
		switch s.Status {
		case SubscriptionStatusPaid:
			hasPaid = true
		case SubscriptionStatusPending:
			hasPending = true
		}
	}
	if hasPaid {
		st := SubscriptionStatusPaid
		return &st, nil
	}
	if hasPending {
		st := SubscriptionStatusPending
		return &st, nil
	}
	return nil, nil
}
