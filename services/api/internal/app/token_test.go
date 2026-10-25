package app_test

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func lower(s string) string { return strings.ToLower(s) }
func upper(s string) string { return strings.ToUpper(s) }

func forgeToken(t *testing.T, secret, userID string) string {
	t.Helper()
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"uid": userID, "sid": "00000000-0000-0000-0000-000000000000", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func noneToken(userID string) string {
	s, _ := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{
		"uid": userID, "sid": "00000000-0000-0000-0000-000000000000", "exp": time.Now().Add(time.Hour).Unix(),
	}).SignedString(jwt.UnsafeAllowNoneSignatureType)
	return s
}
