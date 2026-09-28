package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"time"
)

// Password reset codes: 8 characters from a 32 symbol alphabet without look-alike
// characters (40 bits of entropy), valid 30 minutes, at most 5 guesses. They are
// stored as HMAC-SHA256(serverSecret, userID|code) so a database leak alone does
// not allow offline guessing of the small code space.
const (
	ResetCodeLength  = 8
	ResetCodeTTL     = 30 * time.Minute
	ResetMaxAttempts = 5
	resetAlphabet    = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
)

// GenerateResetCode returns a fresh random code.
func GenerateResetCode() (string, error) {
	buf := make([]byte, ResetCodeLength)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	out := make([]byte, ResetCodeLength)
	for i, b := range buf {
		out[i] = resetAlphabet[int(b)&31] // 32 symbols: no modulo bias
	}
	return string(out), nil
}

// NormalizeResetCode tolerates whitespace, dashes and lower case typed by users.
func NormalizeResetCode(code string) string {
	code = strings.ToUpper(strings.TrimSpace(code))
	code = strings.ReplaceAll(code, "-", "")
	return strings.ReplaceAll(code, " ", "")
}

// HashResetCode derives the stored representation of a code.
func HashResetCode(secret []byte, userID, code string) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte("pwreset|" + userID + "|" + NormalizeResetCode(code)))
	return hex.EncodeToString(mac.Sum(nil))
}

// ResetCodeMatches compares in constant time.
func ResetCodeMatches(secret []byte, userID, code, storedHash string) bool {
	got := HashResetCode(secret, userID, code)
	return hmac.Equal([]byte(got), []byte(storedHash))
}

// ResetUsable reports whether a reset row may still be tried.
func ResetUsable(attempts int, expiresAt time.Time, used bool, now time.Time) bool {
	return !used && attempts < ResetMaxAttempts && now.Before(expiresAt)
}
