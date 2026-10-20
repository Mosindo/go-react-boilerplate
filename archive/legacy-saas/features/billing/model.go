package billing

import (
	"encoding/json"
	"time"
)

type Subscription struct {
	ID                    string     `json:"id"`
	OrganizationID        string     `json:"organizationId"`
	Provider              string     `json:"provider"`
	StripeCustomerID      string     `json:"stripeCustomerId,omitempty"`
	StripeSubscriptionID  string     `json:"stripeSubscriptionId,omitempty"`
	StripeCheckoutSession string     `json:"stripeCheckoutSessionId,omitempty"`
	Status                string     `json:"status"`
	CurrentPeriodStart    *time.Time `json:"currentPeriodStart,omitempty"`
	CurrentPeriodEnd      *time.Time `json:"currentPeriodEnd,omitempty"`
	CancelAtPeriodEnd     bool       `json:"cancelAtPeriodEnd"`
	CanceledAt            *time.Time `json:"canceledAt,omitempty"`
	CreatedAt             time.Time  `json:"createdAt"`
	UpdatedAt             time.Time  `json:"updatedAt"`
}

type CheckoutResponse struct {
	SessionID      string `json:"sessionId"`
	CheckoutURL    string `json:"checkoutUrl"`
	OrganizationID string `json:"organizationId"`
	Status         string `json:"status"`
}

type WebhookResponse struct {
	Received bool `json:"received"`
}

type stripeCustomerResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
}

type stripeCheckoutSessionResponse struct {
	ID               string            `json:"id"`
	URL              string            `json:"url"`
	Customer         string            `json:"customer"`
	Subscription     string            `json:"subscription"`
	Status           string            `json:"status"`
	PaymentStatus    string            `json:"payment_status"`
	ClientRefID      string            `json:"client_reference_id"`
	Metadata         map[string]string `json:"metadata"`
	CustomerEmail    string            `json:"customer_email"`
	ExpiresAtUnix    int64             `json:"expires_at"`
	SubscriptionData struct {
		Metadata map[string]string `json:"metadata"`
	} `json:"subscription_data"`
}

type stripeEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

type stripeSubscriptionObject struct {
	ID                 string            `json:"id"`
	Customer           string            `json:"customer"`
	Metadata           map[string]string `json:"metadata"`
	Status             string            `json:"status"`
	CancelAtPeriodEnd  bool              `json:"cancel_at_period_end"`
	CurrentPeriodStart int64             `json:"current_period_start"`
	CurrentPeriodEnd   int64             `json:"current_period_end"`
	CanceledAtUnix     int64             `json:"canceled_at"`
}
