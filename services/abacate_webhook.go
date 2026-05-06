package services

import (
	"errors"
	"log"
	"strings"
	"time"

	"rose-academy/models"
)

func webhookTrim(s string) string {
	return strings.TrimSpace(s)
}

// webhookPayload alguns payloads v2 envolvem o objeto em "payload".
func webhookPayload(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}
	if p, ok := data["payload"].(map[string]interface{}); ok {
		return p
	}
	return data
}

// WebhookExtractBillingID extrai id de cobrança/checkout (bill_*) do payload Abacate Pay.
func WebhookExtractBillingID(data map[string]interface{}) string {
	data = webhookPayload(data)
	if data == nil {
		return ""
	}
	if b, _ := data["billing"].(map[string]interface{}); b != nil {
		if id, _ := b["id"].(string); id != "" {
			return id
		}
	}
	if ch, _ := data["checkout"].(map[string]interface{}); ch != nil {
		if id, _ := ch["id"].(string); id != "" {
			return id
		}
	}
	if subs, _ := data["subscription"].(map[string]interface{}); subs != nil {
		if ch, _ := subs["checkout"].(map[string]interface{}); ch != nil {
			if id, _ := ch["id"].(string); id != "" && strings.HasPrefix(id, "bill_") {
				return id
			}
		}
	}
	if p, _ := data["payment"].(map[string]interface{}); p != nil {
		if b, _ := p["billing"].(map[string]interface{}); b != nil {
			if id, _ := b["id"].(string); id != "" {
				return id
			}
		}
	}
	if id, _ := data["id"].(string); id != "" && strings.HasPrefix(id, "bill_") {
		return id
	}
	return ""
}

// WebhookExtractSubscriptionID extrai id de assinatura (subs_*) do payload.
func WebhookExtractSubscriptionID(data map[string]interface{}) string {
	data = webhookPayload(data)
	if data == nil {
		return ""
	}
	if s, _ := data["subscription"].(map[string]interface{}); s != nil {
		if id, _ := s["id"].(string); id != "" && strings.HasPrefix(id, "subs_") {
			return id
		}
	}
	if id, _ := data["id"].(string); id != "" && strings.HasPrefix(id, "subs_") {
		return id
	}
	return ""
}

// WebhookExtractCheckoutExternalID externalId igual ao enviado em POST /v2/subscriptions/create.
func WebhookExtractCheckoutExternalID(data map[string]interface{}) string {
	data = webhookPayload(data)
	if data == nil {
		return ""
	}
	if v, ok := data["externalId"].(string); ok && v != "" {
		return v
	}
	if v, ok := data["external_id"].(string); ok && v != "" {
		return v
	}
	if ch, _ := data["checkout"].(map[string]interface{}); ch != nil {
		if v, ok := ch["externalId"].(string); ok && v != "" {
			return v
		}
	}
	return ""
}

// HandleAbacateWebhook aplica eventos documentados em https://docs.abacatepay.com/pages/webhooks/reference
func (s *BillingService) HandleAbacateWebhook(event string, data map[string]interface{}) {
	bill := webhookTrim(WebhookExtractBillingID(data))
	sub := webhookTrim(WebhookExtractSubscriptionID(data))
	ext := webhookTrim(WebhookExtractCheckoutExternalID(data))

	switch event {
	case "billing.paid", "checkout.completed", "transparent.completed", "subscription.completed":
		if err := s.markPaidFromWebhook(bill, sub, ext); err != nil {
			if errors.Is(err, models.ErrSubscriptionNotFoundForBilling) {
				log.Printf("[webhook] %s: nenhuma linha local (bill=%q sub=%q externalId=%q) — confira se o checkout foi criado por POST /abacate/subscriptions/checkout neste ambiente", event, bill, sub, ext)
			} else {
				log.Printf("[webhook] %s: erro ao marcar pago: %v", event, err)
			}
		} else {
			log.Printf("[webhook] %s: assinatura marcada PAID (bill=%q sub=%q externalId=%q)", event, bill, sub, ext)
		}

	case "subscription.cancelled", "subscription.canceled":
		if err := s.markCancelledFromWebhook(bill, sub, ext); err != nil {
			if errors.Is(err, models.ErrSubscriptionNotFoundForBilling) {
				log.Printf("[webhook] %s: nenhuma linha local (bill=%q sub=%q externalId=%q)", event, bill, sub, ext)
			} else {
				log.Printf("[webhook] %s: erro ao cancelar no banco: %v", event, err)
			}
		} else {
			log.Printf("[webhook] %s: assinatura marcada CANCELLED (bill=%q sub=%q)", event, bill, sub)
		}

	case "subscription.renewed":
		now := time.Now()
		if err := s.subRepo.TouchRenewalPaidAt(bill, sub, ext, now); err != nil {
			if errors.Is(err, models.ErrWebhookNoMatchingSubscription) {
				log.Printf("[webhook] subscription.renewed: nenhuma linha local (bill=%q sub=%q externalId=%q)", bill, sub, ext)
			} else {
				log.Printf("[webhook] subscription.renewed: erro: %v", err)
			}
		} else {
			log.Printf("[webhook] subscription.renewed: paid_at atualizado (bill=%q sub=%q)", bill, sub)
		}

	case "checkout.refunded", "transparent.refunded":
		if err := s.markRefundedFromWebhook(bill, sub, ext); err != nil {
			if errors.Is(err, models.ErrSubscriptionNotFoundForBilling) {
				log.Printf("[webhook] %s: nenhuma linha local (bill=%q sub=%q externalId=%q)", event, bill, sub, ext)
			} else {
				log.Printf("[webhook] %s: erro ao marcar reembolso: %v", event, err)
			}
		} else {
			log.Printf("[webhook] %s: assinatura marcada REFUNDED (bill=%q sub=%q)", event, bill, sub)
		}

	default:
		log.Printf("[webhook] evento não mapeado (ignorado): %q", event)
	}
}

func (s *BillingService) markPaidFromWebhook(billID, subID, checkoutExt string) error {
	now := time.Now()
	return s.subRepo.MarkPaidFromWebhook(billID, subID, checkoutExt, now)
}

func (s *BillingService) markCancelledFromWebhook(billID, subID, checkoutExt string) error {
	return s.subRepo.UpdateStatusFromWebhook(billID, subID, checkoutExt, models.SubscriptionStatusCancelled, nil)
}

func (s *BillingService) markRefundedFromWebhook(billID, subID, checkoutExt string) error {
	return s.subRepo.UpdateStatusFromWebhook(billID, subID, checkoutExt, models.SubscriptionStatusRefunded, nil)
}
