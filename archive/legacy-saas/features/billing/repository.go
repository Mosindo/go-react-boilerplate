package billing

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrSubscriptionNotFound = errors.New("subscription not found")
var ErrBillingContextNotFound = errors.New("billing context not found")

type SubscriptionUpsertParams struct {
	OrganizationID        string
	Provider              string
	StripeCustomerID      string
	StripeSubscriptionID  string
	StripeCheckoutSession string
	Status                string
	CurrentPeriodStart    *time.Time
	CurrentPeriodEnd      *time.Time
	CancelAtPeriodEnd     bool
	CanceledAt            *time.Time
}

type StoredSubscription struct {
	ID                    string
	OrganizationID        string
	Provider              string
	StripeCustomerID      string
	StripeSubscriptionID  string
	StripeCheckoutSession string
	Status                string
	CurrentPeriodStart    *time.Time
	CurrentPeriodEnd      *time.Time
	CancelAtPeriodEnd     bool
	CanceledAt            *time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

type BillingContext struct {
	OrganizationID   string
	BillingEmail     string
	StripeCustomerID string
}

type Repository interface {
	UpsertSubscription(ctx context.Context, params SubscriptionUpsertParams) (StoredSubscription, error)
	GetByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (StoredSubscription, error)
	GetByStripeCheckoutSession(ctx context.Context, stripeCheckoutSessionID string) (StoredSubscription, error)
	GetByOrganization(ctx context.Context, organizationID string) (StoredSubscription, error)
	GetBillingContext(ctx context.Context, organizationID, userID string) (BillingContext, error)
}

type PGRepository struct {
	dbPool *pgxpool.Pool
}

func NewPGRepository(dbPool *pgxpool.Pool) *PGRepository {
	return &PGRepository{dbPool: dbPool}
}

func (r *PGRepository) UpsertSubscription(ctx context.Context, params SubscriptionUpsertParams) (StoredSubscription, error) {
	var stored StoredSubscription
	err := r.dbPool.QueryRow(ctx, `
		INSERT INTO subscriptions (
			organization_id,
			provider,
			stripe_customer_id,
			stripe_subscription_id,
			stripe_checkout_session_id,
			status,
			current_period_start,
			current_period_end,
			cancel_at_period_end,
			canceled_at
		)
		VALUES ($1, $2, NULLIF($3, ''), NULLIF($4, ''), NULLIF($5, ''), $6, $7, $8, $9, $10)
		ON CONFLICT (organization_id, provider) DO UPDATE
		SET stripe_customer_id = COALESCE(NULLIF(EXCLUDED.stripe_customer_id, ''), subscriptions.stripe_customer_id),
		    stripe_subscription_id = COALESCE(NULLIF(EXCLUDED.stripe_subscription_id, ''), subscriptions.stripe_subscription_id),
		    stripe_checkout_session_id = COALESCE(NULLIF(EXCLUDED.stripe_checkout_session_id, ''), subscriptions.stripe_checkout_session_id),
		    status = EXCLUDED.status,
		    current_period_start = COALESCE(EXCLUDED.current_period_start, subscriptions.current_period_start),
		    current_period_end = COALESCE(EXCLUDED.current_period_end, subscriptions.current_period_end),
		    cancel_at_period_end = EXCLUDED.cancel_at_period_end,
		    canceled_at = COALESCE(EXCLUDED.canceled_at, subscriptions.canceled_at),
		    updated_at = NOW()
		RETURNING id, organization_id, provider, COALESCE(stripe_customer_id, ''), COALESCE(stripe_subscription_id, ''),
		          COALESCE(stripe_checkout_session_id, ''), status, current_period_start, current_period_end,
		          cancel_at_period_end, canceled_at, created_at, updated_at
	`,
		params.OrganizationID,
		params.Provider,
		params.StripeCustomerID,
		params.StripeSubscriptionID,
		params.StripeCheckoutSession,
		params.Status,
		params.CurrentPeriodStart,
		params.CurrentPeriodEnd,
		params.CancelAtPeriodEnd,
		params.CanceledAt,
	).Scan(
		&stored.ID,
		&stored.OrganizationID,
		&stored.Provider,
		&stored.StripeCustomerID,
		&stored.StripeSubscriptionID,
		&stored.StripeCheckoutSession,
		&stored.Status,
		&stored.CurrentPeriodStart,
		&stored.CurrentPeriodEnd,
		&stored.CancelAtPeriodEnd,
		&stored.CanceledAt,
		&stored.CreatedAt,
		&stored.UpdatedAt,
	)
	if err != nil {
		return StoredSubscription{}, err
	}
	return stored, nil
}

func (r *PGRepository) GetByStripeSubscriptionID(ctx context.Context, stripeSubscriptionID string) (StoredSubscription, error) {
	return r.getOne(ctx, `
		SELECT id, organization_id, provider, COALESCE(stripe_customer_id, ''), COALESCE(stripe_subscription_id, ''),
		       COALESCE(stripe_checkout_session_id, ''), status, current_period_start, current_period_end,
		       cancel_at_period_end, canceled_at, created_at, updated_at
		FROM subscriptions
		WHERE stripe_subscription_id = $1
	`, stripeSubscriptionID)
}

func (r *PGRepository) GetByStripeCheckoutSession(ctx context.Context, stripeCheckoutSessionID string) (StoredSubscription, error) {
	return r.getOne(ctx, `
		SELECT id, organization_id, provider, COALESCE(stripe_customer_id, ''), COALESCE(stripe_subscription_id, ''),
		       COALESCE(stripe_checkout_session_id, ''), status, current_period_start, current_period_end,
		       cancel_at_period_end, canceled_at, created_at, updated_at
		FROM subscriptions
		WHERE stripe_checkout_session_id = $1
	`, stripeCheckoutSessionID)
}

func (r *PGRepository) GetByOrganization(ctx context.Context, organizationID string) (StoredSubscription, error) {
	return r.getOne(ctx, `
		SELECT id, organization_id, provider, COALESCE(stripe_customer_id, ''), COALESCE(stripe_subscription_id, ''),
		       COALESCE(stripe_checkout_session_id, ''), status, current_period_start, current_period_end,
		       cancel_at_period_end, canceled_at, created_at, updated_at
		FROM subscriptions
		WHERE organization_id = $1
		  AND provider = 'stripe'
	`, organizationID)
}

func (r *PGRepository) GetBillingContext(ctx context.Context, organizationID, userID string) (BillingContext, error) {
	var billing BillingContext
	err := r.dbPool.QueryRow(ctx, `
		SELECT
			u.organization_id,
			u.email,
			COALESCE(s.stripe_customer_id, '')
		FROM users u
		LEFT JOIN subscriptions s
		  ON s.organization_id = u.organization_id
		 AND s.provider = 'stripe'
		WHERE u.id = $1
		  AND u.organization_id = $2
		LIMIT 1
	`, userID, organizationID).Scan(
		&billing.OrganizationID,
		&billing.BillingEmail,
		&billing.StripeCustomerID,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return BillingContext{}, ErrBillingContextNotFound
		}
		return BillingContext{}, err
	}
	return billing, nil
}

func (r *PGRepository) getOne(ctx context.Context, query, value string) (StoredSubscription, error) {
	var stored StoredSubscription
	err := r.dbPool.QueryRow(ctx, query, value).Scan(
		&stored.ID,
		&stored.OrganizationID,
		&stored.Provider,
		&stored.StripeCustomerID,
		&stored.StripeSubscriptionID,
		&stored.StripeCheckoutSession,
		&stored.Status,
		&stored.CurrentPeriodStart,
		&stored.CurrentPeriodEnd,
		&stored.CancelAtPeriodEnd,
		&stored.CanceledAt,
		&stored.CreatedAt,
		&stored.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return StoredSubscription{}, ErrSubscriptionNotFound
		}
		return StoredSubscription{}, err
	}
	return stored, nil
}
