package billing

import (
	"errors"
	"io"
	"net/http"
	"strings"

	"example.com/api/internal/platform/logger"
	"github.com/gin-gonic/gin"
)

const maxWebhookPayloadBytes = 1 << 20

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Checkout(c *gin.Context) {
	organizationID := c.GetString("organizationID")
	userID := c.GetString("userID")
	response, err := h.service.Checkout(c.Request.Context(), organizationID, userID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrOrganizationRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "organization context required"})
		case errors.Is(err, ErrBillingContextNotFound):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "billing user context required"})
		case errors.Is(err, ErrBillingUnavailable):
			status = http.StatusServiceUnavailable
			c.JSON(status, gin.H{"error": "billing is not configured"})
		case errors.Is(err, ErrCheckoutSessionFailed):
			status = http.StatusBadGateway
			c.JSON(status, gin.H{"error": "could not create checkout session"})
		default:
			c.JSON(status, gin.H{"error": "could not start checkout"})
		}
		logger.LogHandlerError(c, "billing.checkout", status, err)
		return
	}

	c.JSON(http.StatusCreated, response)
	logger.LogHandlerEvent(c, "billing.checkout.success", http.StatusCreated, map[string]string{
		"organization_id": response.OrganizationID,
		"session_id":      response.SessionID,
	})
}

func (h *Handler) Subscription(c *gin.Context) {
	organizationID := strings.TrimSpace(c.GetString("organizationID"))
	subscription, err := h.service.GetSubscription(c.Request.Context(), organizationID)
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrOrganizationRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "organization context required"})
		default:
			c.JSON(status, gin.H{"error": "could not load subscription"})
		}
		logger.LogHandlerError(c, "billing.subscription", status, err)
		return
	}

	c.JSON(http.StatusOK, subscription)
}

func (h *Handler) Webhook(c *gin.Context) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxWebhookPayloadBytes)
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		logger.LogHandlerError(c, "billing.webhook.read", http.StatusBadRequest, err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid webhook payload"})
		return
	}

	received, err := h.service.ProcessWebhook(c.Request.Context(), payload, c.GetHeader("Stripe-Signature"))
	if err != nil {
		status := http.StatusInternalServerError
		switch {
		case errors.Is(err, ErrWebhookUnavailable):
			status = http.StatusServiceUnavailable
			c.JSON(status, gin.H{"error": "billing webhook is not configured"})
		case errors.Is(err, ErrInvalidWebhookSignature):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "invalid stripe signature"})
		case errors.Is(err, ErrOrganizationRequired):
			status = http.StatusBadRequest
			c.JSON(status, gin.H{"error": "organization context required"})
		default:
			c.JSON(status, gin.H{"error": "could not process billing webhook"})
		}
		logger.LogHandlerError(c, "billing.webhook", status, err)
		return
	}

	c.JSON(http.StatusOK, WebhookResponse{Received: received})
}

func (h *Handler) SuccessPage(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Checkout complete</title>
  <style>
    body { margin: 0; background: #0b1020; color: #f7f8fb; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    main { min-height: 100vh; display: grid; place-items: center; padding: 24px; }
    section { max-width: 560px; background: #121a31; border: 1px solid #27324e; border-radius: 20px; padding: 32px; box-shadow: 0 30px 80px rgba(0,0,0,0.35); }
    h1 { margin: 0 0 12px; font-size: 28px; }
    p { margin: 0 0 10px; line-height: 1.6; color: #c6d0e6; }
    code { color: #ffffff; }
  </style>
</head>
<body>
  <main>
    <section>
      <h1>Subscription confirmed</h1>
      <p>Your Stripe Checkout flow completed successfully.</p>
      <p>You can now return to <strong>go-react-boilerplate</strong>. The app will refresh your billing status automatically.</p>
      <p>You can safely close this page after returning to the app.</p>
    </section>
  </main>
</body>
</html>`))
}

func (h *Handler) CancelPage(c *gin.Context) {
	c.Data(http.StatusOK, "text/html; charset=utf-8", []byte(`<!doctype html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Checkout canceled</title>
  <style>
    body { margin: 0; background: #0b1020; color: #f7f8fb; font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif; }
    main { min-height: 100vh; display: grid; place-items: center; padding: 24px; }
    section { max-width: 560px; background: #121a31; border: 1px solid #27324e; border-radius: 20px; padding: 32px; box-shadow: 0 30px 80px rgba(0,0,0,0.35); }
    h1 { margin: 0 0 12px; font-size: 28px; }
    p { margin: 0 0 10px; line-height: 1.6; color: #c6d0e6; }
  </style>
</head>
<body>
  <main>
    <section>
      <h1>Checkout canceled</h1>
      <p>No charge was applied.</p>
      <p>You can return to <strong>go-react-boilerplate</strong> and restart the billing flow whenever you are ready.</p>
    </section>
  </main>
</body>
</html>`))
}
