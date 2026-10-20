package billing

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

var (
	ErrBillingUnavailable      = errors.New("billing is not configured")
	ErrWebhookUnavailable      = errors.New("billing webhook is not configured")
	ErrOrganizationRequired    = errors.New("organization id required")
	ErrCheckoutSessionFailed   = errors.New("stripe checkout session creation failed")
	ErrInvalidWebhookSignature = errors.New("invalid stripe webhook signature")
)

type HTTPDoer interface {
	Do(req *http.Request) (*http.Response, error)
}

type Config struct {
	StripeSecretKey     string
	StripeWebhookSecret string
	StripePriceID       string
	AppBaseURL          string
	HTTPClient          HTTPDoer
}

type Service struct {
	repo                Repository
	stripeSecretKey     string
	stripeWebhookSecret string
	stripePriceID       string
	appBaseURL          string
	httpClient          HTTPDoer
}

const (
	stripeAPIBaseURL     = "https://api.stripe.com/v1"
	defaultProvider      = "stripe"
	webhookTimestampSkew = 5 * time.Minute
)

func NewService(repo Repository, cfg Config) *Service {
	httpClient := cfg.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 10 * time.Second}
	}
	return &Service{
		repo:                repo,
		stripeSecretKey:     strings.TrimSpace(cfg.StripeSecretKey),
		stripeWebhookSecret: strings.TrimSpace(cfg.StripeWebhookSecret),
		stripePriceID:       strings.TrimSpace(cfg.StripePriceID),
		appBaseURL:          strings.TrimSpace(strings.TrimRight(cfg.AppBaseURL, "/")),
		httpClient:          httpClient,
	}
}

func (s *Service) Checkout(ctx context.Context, organizationID, userID string) (CheckoutResponse, error) {
	if strings.TrimSpace(organizationID) == "" {
		return CheckoutResponse{}, ErrOrganizationRequired
	}
	if !s.checkoutConfigured() {
		return CheckoutResponse{}, ErrBillingUnavailable
	}
	if strings.TrimSpace(userID) == "" {
		return CheckoutResponse{}, ErrOrganizationRequired
	}

	billingContext, err := s.repo.GetBillingContext(ctx, organizationID, userID)
	if err != nil {
		return CheckoutResponse{}, err
	}

	customerID := billingContext.StripeCustomerID
	if customerID == "" {
		customerID, err = s.createStripeCustomer(ctx, billingContext.BillingEmail, organizationID)
		if err != nil {
			return CheckoutResponse{}, err
		}
	}

	form := url.Values{}
	form.Set("mode", "subscription")
	form.Set("success_url", s.appBaseURL+"/billing/success?session_id={CHECKOUT_SESSION_ID}")
	form.Set("cancel_url", s.appBaseURL+"/billing/cancel")
	form.Set("line_items[0][price]", s.stripePriceID)
	form.Set("line_items[0][quantity]", "1")
	form.Set("client_reference_id", organizationID)
	form.Set("metadata[organization_id]", organizationID)
	form.Set("subscription_data[metadata][organization_id]", organizationID)
	form.Set("customer", customerID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stripeAPIBaseURL+"/checkout/sessions", strings.NewReader(form.Encode()))
	if err != nil {
		return CheckoutResponse{}, err
	}
	req.Header.Set("Authorization", "Bearer "+s.stripeSecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return CheckoutResponse{}, fmt.Errorf("%w: request failed", ErrCheckoutSessionFailed)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return CheckoutResponse{}, fmt.Errorf("%w: response unreadable", ErrCheckoutSessionFailed)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return CheckoutResponse{}, fmt.Errorf("%w: status=%d", ErrCheckoutSessionFailed, resp.StatusCode)
	}

	var session stripeCheckoutSessionResponse
	if err := json.Unmarshal(body, &session); err != nil {
		return CheckoutResponse{}, fmt.Errorf("%w: invalid stripe response", ErrCheckoutSessionFailed)
	}

	storedCustomerID := coalesce(session.Customer, customerID)
	if _, err := s.repo.UpsertSubscription(ctx, SubscriptionUpsertParams{
		OrganizationID:        organizationID,
		Provider:              defaultProvider,
		StripeCustomerID:      storedCustomerID,
		StripeSubscriptionID:  session.Subscription,
		StripeCheckoutSession: session.ID,
		Status:                coalesce(session.Status, "checkout_open"),
	}); err != nil {
		return CheckoutResponse{}, err
	}

	return CheckoutResponse{
		SessionID:      session.ID,
		CheckoutURL:    session.URL,
		OrganizationID: organizationID,
		Status:         coalesce(session.Status, "checkout_open"),
	}, nil
}

func (s *Service) GetSubscription(ctx context.Context, organizationID string) (Subscription, error) {
	if strings.TrimSpace(organizationID) == "" {
		return Subscription{}, ErrOrganizationRequired
	}

	stored, err := s.repo.GetByOrganization(ctx, organizationID)
	if err != nil {
		if errors.Is(err, ErrSubscriptionNotFound) {
			return Subscription{
				OrganizationID: organizationID,
				Provider:       defaultProvider,
				Status:         "inactive",
			}, nil
		}
		return Subscription{}, err
	}

	return Subscription{
		ID:                    stored.ID,
		OrganizationID:        stored.OrganizationID,
		Provider:              stored.Provider,
		StripeCustomerID:      stored.StripeCustomerID,
		StripeSubscriptionID:  stored.StripeSubscriptionID,
		StripeCheckoutSession: stored.StripeCheckoutSession,
		Status:                stored.Status,
		CurrentPeriodStart:    stored.CurrentPeriodStart,
		CurrentPeriodEnd:      stored.CurrentPeriodEnd,
		CancelAtPeriodEnd:     stored.CancelAtPeriodEnd,
		CanceledAt:            stored.CanceledAt,
		CreatedAt:             stored.CreatedAt,
		UpdatedAt:             stored.UpdatedAt,
	}, nil
}

func (s *Service) ProcessWebhook(ctx context.Context, payload []byte, signatureHeader string) (bool, error) {
	if strings.TrimSpace(s.stripeWebhookSecret) == "" {
		return false, ErrWebhookUnavailable
	}
	if err := verifyStripeSignature(payload, signatureHeader, s.stripeWebhookSecret, time.Now().UTC()); err != nil {
		return false, err
	}

	var event stripeEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return false, err
	}

	switch event.Type {
	case "checkout.session.completed":
		return true, s.handleCheckoutCompleted(ctx, event.Data.Object)
	case "customer.subscription.created", "customer.subscription.updated", "customer.subscription.deleted":
		return true, s.handleSubscriptionUpdated(ctx, event.Data.Object)
	default:
		return true, nil
	}
}

func (s *Service) handleCheckoutCompleted(ctx context.Context, raw json.RawMessage) error {
	var session stripeCheckoutSessionResponse
	if err := json.Unmarshal(raw, &session); err != nil {
		return err
	}

	organizationID := stripeOrganizationID(session.Metadata, session.ClientRefID)
	if organizationID == "" && session.ID != "" {
		stored, err := s.repo.GetByStripeCheckoutSession(ctx, session.ID)
		if err == nil {
			organizationID = stored.OrganizationID
		}
	}
	if organizationID == "" {
		return ErrOrganizationRequired
	}

	_, err := s.repo.UpsertSubscription(ctx, SubscriptionUpsertParams{
		OrganizationID:        organizationID,
		Provider:              defaultProvider,
		StripeCustomerID:      session.Customer,
		StripeSubscriptionID:  session.Subscription,
		StripeCheckoutSession: session.ID,
		Status:                coalesce(session.PaymentStatus, "checkout_completed"),
	})
	return err
}

func (s *Service) handleSubscriptionUpdated(ctx context.Context, raw json.RawMessage) error {
	var sub stripeSubscriptionObject
	if err := json.Unmarshal(raw, &sub); err != nil {
		return err
	}

	organizationID := stripeOrganizationID(sub.Metadata, "")
	if organizationID == "" && sub.ID != "" {
		stored, err := s.repo.GetByStripeSubscriptionID(ctx, sub.ID)
		if err == nil {
			organizationID = stored.OrganizationID
		}
	}
	if organizationID == "" {
		return ErrOrganizationRequired
	}

	_, err := s.repo.UpsertSubscription(ctx, SubscriptionUpsertParams{
		OrganizationID:       organizationID,
		Provider:             defaultProvider,
		StripeCustomerID:     sub.Customer,
		StripeSubscriptionID: sub.ID,
		Status:               coalesce(sub.Status, "active"),
		CurrentPeriodStart:   unixPtr(sub.CurrentPeriodStart),
		CurrentPeriodEnd:     unixPtr(sub.CurrentPeriodEnd),
		CancelAtPeriodEnd:    sub.CancelAtPeriodEnd,
		CanceledAt:           unixPtr(sub.CanceledAtUnix),
	})
	return err
}

func (s *Service) checkoutConfigured() bool {
	return s.stripeSecretKey != "" && s.stripePriceID != "" && s.appBaseURL != ""
}

func (s *Service) createStripeCustomer(ctx context.Context, email, organizationID string) (string, error) {
	form := url.Values{}
	form.Set("email", strings.TrimSpace(email))
	form.Set("metadata[organization_id]", organizationID)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, stripeAPIBaseURL+"/customers", strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+s.stripeSecretKey)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("%w: request failed", ErrCheckoutSessionFailed)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("%w: response unreadable", ErrCheckoutSessionFailed)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("%w: status=%d", ErrCheckoutSessionFailed, resp.StatusCode)
	}

	var customer stripeCustomerResponse
	if err := json.Unmarshal(body, &customer); err != nil {
		return "", fmt.Errorf("%w: invalid stripe customer response", ErrCheckoutSessionFailed)
	}
	if strings.TrimSpace(customer.ID) == "" {
		return "", fmt.Errorf("%w: missing stripe customer id", ErrCheckoutSessionFailed)
	}

	return customer.ID, nil
}

func verifyStripeSignature(payload []byte, header, secret string, now time.Time) error {
	if strings.TrimSpace(header) == "" || strings.TrimSpace(secret) == "" {
		return ErrInvalidWebhookSignature
	}

	var timestamp int64
	signatures := make([]string, 0)
	for _, part := range strings.Split(header, ",") {
		keyValue := strings.SplitN(strings.TrimSpace(part), "=", 2)
		if len(keyValue) != 2 {
			continue
		}
		switch keyValue[0] {
		case "t":
			parsed, err := strconv.ParseInt(keyValue[1], 10, 64)
			if err != nil {
				return ErrInvalidWebhookSignature
			}
			timestamp = parsed
		case "v1":
			signatures = append(signatures, keyValue[1])
		}
	}

	if timestamp == 0 || len(signatures) == 0 {
		return ErrInvalidWebhookSignature
	}
	if absDuration(now.Sub(time.Unix(timestamp, 0).UTC())) > webhookTimestampSkew {
		return ErrInvalidWebhookSignature
	}

	signedPayload := strconv.FormatInt(timestamp, 10) + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(signedPayload))
	expected := hex.EncodeToString(mac.Sum(nil))

	for _, candidate := range signatures {
		if subtle.ConstantTimeCompare([]byte(expected), []byte(candidate)) == 1 {
			return nil
		}
	}

	return ErrInvalidWebhookSignature
}

func stripeOrganizationID(metadata map[string]string, fallback string) string {
	if metadata != nil {
		if orgID := strings.TrimSpace(metadata["organization_id"]); orgID != "" {
			return orgID
		}
	}
	return strings.TrimSpace(fallback)
}

func coalesce(value, fallback string) string {
	if strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func unixPtr(value int64) *time.Time {
	if value <= 0 {
		return nil
	}
	t := time.Unix(value, 0).UTC()
	return &t
}

func absDuration(value time.Duration) time.Duration {
	if value < 0 {
		return -value
	}
	return value
}

type stubHTTPResponse struct {
	StatusCode int
	Body       string
}

func newJSONResponse(statusCode int, body string) *http.Response {
	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}
