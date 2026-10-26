package main

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

func signTestToken(t *testing.T, secret, userID string, exp time.Time) string {
	t.Helper()
	claims := jwt.MapClaims{"uid": userID, "sid": "00000000-0000-0000-0000-000000000000"}
	if !exp.IsZero() {
		claims["exp"] = exp.Unix()
	}
	s, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatal(err)
	}
	return s
}
