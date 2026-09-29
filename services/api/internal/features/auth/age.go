package auth

import (
	"errors"
	"time"
)

const (
	// MinAge is the legal minimum age to use the service.
	MinAge = 18
	// MaxAge is the largest age accepted at registration (anything above is treated as bogus input).
	MaxAge = 120
)

var (
	ErrBirthDateFuture = errors.New("birth date cannot be in the future")
	ErrBirthDateAbsurd = errors.New("birth date is not plausible")
	ErrUnderage        = errors.New("you must be at least 18 years old")
)

// AgeOn returns the completed years between birth and now, comparing calendar dates in UTC.
// The result is negative when birth is after now.
func AgeOn(birth, now time.Time) int {
	b, n := birth.UTC(), now.UTC()
	age := n.Year() - b.Year()
	if n.Month() < b.Month() || (n.Month() == b.Month() && n.Day() < b.Day()) {
		age--
	}
	return age
}

// ValidateBirthDate enforces the age gate: not in the future, at most MaxAge, at least MinAge.
func ValidateBirthDate(birth, now time.Time) error {
	b, n := birth.UTC(), now.UTC()
	by, bm, bd := b.Date()
	ny, nm, nd := n.Date()
	if time.Date(by, bm, bd, 0, 0, 0, 0, time.UTC).After(time.Date(ny, nm, nd, 0, 0, 0, 0, time.UTC)) {
		return ErrBirthDateFuture
	}
	age := AgeOn(birth, now)
	if age > MaxAge {
		return ErrBirthDateAbsurd
	}
	if age < MinAge {
		return ErrUnderage
	}
	return nil
}
