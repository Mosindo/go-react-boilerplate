package billing

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

type stubRepository struct {
	upserted            []SubscriptionUpsertParams
	subscriptionByID    map[string]StoredSubscription
	checkoutSessionByID map[string]StoredSubscription
	organizationByID    map[string]StoredSubscription
	billingContexts     map[string]BillingContext
}

func (r *stubRepository) UpsertSubscription(_ context.Context, params SubscriptionUpsertParams) (StoredSubscription, error) {
	r.upserted = append(r.upserted, params)
	return StoredSubscription{
		ID:                    "sub-row-1",
		OrganizationID:        params.OrganizationID,
		Provider:              params.Provider,
		StripeCustomerID:      params.StripeCustomerID,
		StripeSubscriptionID:  params.StripeSubscriptionID,
		StripeCheckoutSession: params.StripeCheckoutSession,
		Status:                params.Status,
	}, nil
}

func (r *stubRepository) GetByStripeSubscriptionID(_ context.Context, stripeSubscriptionID string) (StoredSubscription, error) {
	if stored, ok := r.subscriptionByID[stripeSubscriptionID]; ok {
		return stored, nil
	}
	return StoredSubscription{}, ErrSubscriptionNotFound
}

func (r *stubRepository) GetByStripeCheckoutSession(_ context.Context, stripeCheckoutSessionID string) (StoredSubscription, error) {
	if stored, ok := r.checkoutSessionByID[stripeCheckoutSessionID]; ok {
		return stored, nil
	}
	return StoredSubscription{}, ErrSubscriptionNotFound
}

func (r *stubRepository) GetByOrganization(_ context.Context, organizationID string) (StoredSubscription, error) {
	if stored, ok := r.organizationByID[organizationID]; ok {
		return stored, nil
	}
	return StoredSubscription{}, ErrSubscriptionNotFound
}

func (r *stubRepository) GetBillingContext(_ context.Context, organizationID, userID string) (BillingContext, error) {
	if ctx, ok := r.billingContexts[organizationID+":"+userID]; ok {
		return ctx, nil
	}
	return BillingContext{}, ErrBillingContextNotFound
}

type capturedRequest struct {
	URL    string
	Method string
	Body   string
}

type stubDoer struct {
	responses []*http.Response
	err       error
	requests  []capturedRequest
}

func (d *stubDoer) Do(req *http.Request) (*http.Response, error) {
	if d.err != nil {
		return nil, d.err
	}
	body, _ := io.ReadAll(req.Body)
	d.requests = append(d.requests, capturedRequest{
		URL:    req.URL.String(),
		Method: req.Method,
		Body:   string(body),
	})
	if len(d.responses) == 0 {
		return nil, errors.New("no stub response configured")
	}
	response := d.responses[0]
	d.responses = d.responses[1:]
	return response, nil
}

func TestCheckoutRequiresConfiguration(t *testing.T) {
	repo := &stubRepository{}
	svc := NewService(repo, Config{})

	_, err := svc.Checkout(context.Background(), "org-1", "user-1")
	if !errors.Is(err, ErrBillingUnavailable) {
		t.Fatalf("expected ErrBillingUnavailable, got %v", err)
	}
}

func TestCheckoutCreatesCustomerAndPendingSubscription(t *testing.T) {
	repo := &stubRepository{
		billingContexts: map[string]BillingContext{
			"org-1:user-1": {
				OrganizationID: "org-1",
				BillingEmail:   "owner@example.com",
			},
		},
	}
	doer := &stubDoer{
		responses: []*http.Response{
			newJSONResponse(http.StatusOK, `{"id":"cus_test_123","email":"owner@example.com"}`),
			newJSONResponse(http.StatusOK, `{"id":"cs_test_123","url":"https://checkout.stripe.test/session/cs_test_123","customer":"cus_test_123","status":"open"}`),
		},
	}
	svc := NewService(repo, Config{
		StripeSecretKey: "sk_test_123",
		StripePriceID:   "price_test_123",
		AppBaseURL:      "https://app.example.com",
		HTTPClient:      doer,
	})

	response, err := svc.Checkout(context.Background(), "org-1", "user-1")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if response.SessionID != "cs_test_123" {
		t.Fatalf("expected checkout session id, got %+v", response)
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("expected one upsert, got %d", len(repo.upserted))
	}
	if repo.upserted[0].OrganizationID != "org-1" {
		t.Fatalf("expected organization id org-1, got %q", repo.upserted[0].OrganizationID)
	}
	if repo.upserted[0].StripeCustomerID != "cus_test_123" {
		t.Fatalf("expected customer id cus_test_123, got %q", repo.upserted[0].StripeCustomerID)
	}
	if len(doer.requests) != 2 {
		t.Fatalf("expected 2 outgoing stripe requests, got %d", len(doer.requests))
	}
	if !strings.HasSuffix(doer.requests[0].URL, "/customers") {
		t.Fatalf("expected first request to /customers, got %q", doer.requests[0].URL)
	}
	if !strings.Contains(doer.requests[1].Body, "customer=cus_test_123") {
		t.Fatalf("expected checkout request to include customer id, got body %q", doer.requests[1].Body)
	}
}

func TestCheckoutReusesExistingCustomer(t *testing.T) {
	repo := &stubRepository{
		billingContexts: map[string]BillingContext{
			"org-1:user-1": {
				OrganizationID:   "org-1",
				BillingEmail:     "owner@example.com",
				StripeCustomerID: "cus_existing",
			},
		},
	}
	doer := &stubDoer{
		responses: []*http.Response{
			newJSONResponse(http.StatusOK, `{"id":"cs_test_123","url":"https://checkout.stripe.test/session/cs_test_123","customer":"cus_existing","status":"open"}`),
		},
	}
	svc := NewService(repo, Config{
		StripeSecretKey: "sk_test_123",
		StripePriceID:   "price_test_123",
		AppBaseURL:      "https://app.example.com",
		HTTPClient:      doer,
	})

	_, err := svc.Checkout(context.Background(), "org-1", "user-1")
	if err != nil {
		t.Fatalf("checkout: %v", err)
	}
	if len(doer.requests) != 1 {
		t.Fatalf("expected one outgoing stripe request, got %d", len(doer.requests))
	}
	if !strings.HasSuffix(doer.requests[0].URL, "/checkout/sessions") {
		t.Fatalf("expected checkout session request, got %q", doer.requests[0].URL)
	}
}

func TestProcessWebhookRejectsInvalidSignature(t *testing.T) {
	repo := &stubRepository{}
	svc := NewService(repo, Config{StripeWebhookSecret: "whsec_test"})

	_, err := svc.ProcessWebhook(context.Background(), []byte(`{"id":"evt_test","type":"customer.subscription.updated"}`), "t=1,v1=bad")
	if !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("expected ErrInvalidWebhookSignature, got %v", err)
	}
}

func TestVerifyStripeSignatureRejectsFutureTimestampOutsideSkew(t *testing.T) {
	payload := []byte(`{"id":"evt_future"}`)
	now := time.Now().UTC()
	signature := signTestPayload(payload, "whsec_test", now.Add(10*time.Minute))
	if err := verifyStripeSignature(payload, signature, "whsec_test", now); !errors.Is(err, ErrInvalidWebhookSignature) {
		t.Fatalf("expected ErrInvalidWebhookSignature for future timestamp, got %v", err)
	}
}

func TestProcessWebhookUpdatesSubscription(t *testing.T) {
	repo := &stubRepository{}
	svc := NewService(repo, Config{StripeWebhookSecret: "whsec_test"})

	payload := []byte(`{"id":"evt_1","type":"customer.subscription.updated","data":{"object":{"id":"sub_123","customer":"cus_123","metadata":{"organization_id":"org_123"},"status":"active","cancel_at_period_end":false,"current_period_start":1735689600,"current_period_end":1738368000}}}`)
	signature := signTestPayload(payload, "whsec_test", time.Now())

	received, err := svc.ProcessWebhook(context.Background(), payload, signature)
	if err != nil {
		t.Fatalf("process webhook: %v", err)
	}
	if !received {
		t.Fatalf("expected received=true")
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("expected one upsert, got %d", len(repo.upserted))
	}
	if repo.upserted[0].OrganizationID != "org_123" {
		t.Fatalf("expected org_123, got %q", repo.upserted[0].OrganizationID)
	}
	if repo.upserted[0].StripeSubscriptionID != "sub_123" {
		t.Fatalf("expected sub_123, got %q", repo.upserted[0].StripeSubscriptionID)
	}
}

func TestProcessWebhookFallsBackToStoredCheckoutOrganization(t *testing.T) {
	repo := &stubRepository{
		checkoutSessionByID: map[string]StoredSubscription{
			"cs_fallback": {
				OrganizationID: "org_fallback",
			},
		},
	}
	svc := NewService(repo, Config{StripeWebhookSecret: "whsec_test"})

	payload := []byte(`{"id":"evt_2","type":"checkout.session.completed","data":{"object":{"id":"cs_fallback","customer":"cus_123","subscription":"sub_123","status":"complete","payment_status":"paid"}}}`)
	signature := signTestPayload(payload, "whsec_test", time.Now())

	received, err := svc.ProcessWebhook(context.Background(), payload, signature)
	if err != nil {
		t.Fatalf("process webhook: %v", err)
	}
	if !received {
		t.Fatalf("expected received=true")
	}
	if len(repo.upserted) != 1 {
		t.Fatalf("expected one upsert, got %d", len(repo.upserted))
	}
	if repo.upserted[0].OrganizationID != "org_fallback" {
		t.Fatalf("expected fallback org, got %q", repo.upserted[0].OrganizationID)
	}
	if repo.upserted[0].Status != "paid" {
		t.Fatalf("expected paid status, got %q", repo.upserted[0].Status)
	}
}

func TestGetSubscriptionDefaultsToInactive(t *testing.T) {
	repo := &stubRepository{}
	svc := NewService(repo, Config{})

	subscription, err := svc.GetSubscription(context.Background(), "org-1")
	if err != nil {
		t.Fatalf("get subscription: %v", err)
	}
	if subscription.Status != "inactive" {
		t.Fatalf("expected inactive status, got %q", subscription.Status)
	}
	if subscription.OrganizationID != "org-1" {
		t.Fatalf("expected organization id org-1, got %q", subscription.OrganizationID)
	}
}

func signTestPayload(payload []byte, secret string, now time.Time) string {
	timestamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "." + string(payload)))
	signature := hex.EncodeToString(mac.Sum(nil))
	return strings.Join([]string{"t=" + timestamp, "v1=" + signature}, ",")
}
