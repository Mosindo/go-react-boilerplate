package auth

import (
	"strings"
	"testing"
	"time"

	"example.com/api/internal/platform/middleware"
	"github.com/golang-jwt/jwt/v5"
)

func TestNormalizeEmail(t *testing.T) {
	if got := normalizeEmail("  Jane.DOE@Example.COM \n"); got != "jane.doe@example.com" {
		t.Fatalf("unexpected normalization: %q", got)
	}
}

func TestResetCodeAlphabetAndLength(t *testing.T) {
	seen := map[string]struct{}{}
	for i := 0; i < 200; i++ {
		code, err := generateResetCode()
		if err != nil {
			t.Fatal(err)
		}
		if len(code) != 10 {
			t.Fatalf("expected 10 characters, got %q", code)
		}
		if strings.ContainsAny(code, "01OILl") {
			t.Fatalf("code contains look-alike characters: %q", code)
		}
		seen[code] = struct{}{}
	}
	if len(seen) < 199 {
		t.Fatal("reset codes must be unpredictable")
	}
}

func TestAccessTokenIsHS256WithExpiry(t *testing.T) {
	secret := []byte("unit-test-secret-0123456789abcdef")
	raw, err := signAccessToken(secret, User{ID: "u1", OrganizationID: "o1"}, "s1")
	if err != nil {
		t.Fatal(err)
	}
	claims, err := middleware.ParseAccessToken(secret, raw)
	if err != nil {
		t.Fatalf("token must validate: %v", err)
	}
	exp, _ := claims.GetExpirationTime()
	if exp == nil || time.Until(exp.Time) > accessTokenTTL || time.Until(exp.Time) <= 0 {
		t.Fatalf("expiration must be set and short-lived, got %v", exp)
	}
	if claims["uid"] != "u1" || claims["sid"] != "s1" {
		t.Fatalf("unexpected claims %v", claims)
	}
	if _, err := middleware.ParseAccessToken([]byte("another-secret-0123456789abcdef!"), raw); err == nil {
		t.Fatal("token signed with another secret must be rejected")
	}

	// alg=none and HS512 tokens must be rejected
	none := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()})
	noneRaw, _ := none.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if _, err := middleware.ParseAccessToken(secret, noneRaw); err == nil {
		t.Fatal("alg=none must be rejected")
	}
	hs512, _ := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{"uid": "u1", "sid": "s1", "exp": time.Now().Add(time.Minute).Unix()}).SignedString(secret)
	if _, err := middleware.ParseAccessToken(secret, hs512); err == nil {
		t.Fatal("only HS256 is accepted")
	}
	noExp, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"uid": "u1", "sid": "s1"}).SignedString(secret)
	if _, err := middleware.ParseAccessToken(secret, noExp); err == nil {
		t.Fatal("tokens without expiration must be rejected")
	}
}

func TestValidEmail(t *testing.T) {
	for email, want := range map[string]bool{
		"jane@example.com":        true,
		"jane.doe+tag@sub.ex.org": true,
		"jane@localhost":          false,
		"Jane <jane@example.com>": false,
		"jane@example.":           false,
		"@example.com":            false,
		"jane example@x.com":      false,
		"":                        false,
	} {
		if got := validEmail(email); got != want {
			t.Errorf("validEmail(%q) = %v, want %v", email, got, want)
		}
	}
}
