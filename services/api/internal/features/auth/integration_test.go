package auth

import (
	"context"
	"testing"

	"example.com/api/internal/platform/middleware"
)

// The service signs tokens that the HTTP middleware must accept, and nothing else.
func TestSignedAccessTokenIsAcceptedByMiddleware(t *testing.T) {
	secret := []byte("0123456789abcdef0123456789abcdef")
	svc := NewService(nil, secret, nil)

	token, err := svc.signAccessToken("user-1", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	id, err := middleware.ParseAccessToken(context.Background(), secret, token, nil)
	if err != nil || id.UserID != "user-1" || id.SessionID != "session-1" {
		t.Fatalf("token rejected: %v %+v", err, id)
	}
	if _, err := middleware.ParseAccessToken(context.Background(), []byte("another-secret-another-secret-12"), token, nil); err == nil {
		t.Fatal("token must not validate with another secret")
	}
}
