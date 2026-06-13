// Package auth holds password hashing, JWT issuing/verification, and refresh-token
// generation for the Prekaj-Zeiterfassung backend. (HTTP middleware lives in internal/server.)
package auth

import (
	"errors"

	"golang.org/x/crypto/bcrypt"
)

// ErrPasswordTooLong is returned when a password exceeds bcrypt's 72-byte limit.
var ErrPasswordTooLong = errors.New("auth: password exceeds 72 bytes")

// Hasher hashes and verifies passwords with bcrypt at a configured cost.
type Hasher struct{ cost int }

// NewHasher clamps cost into a safe range (project policy: >= 12).
func NewHasher(cost int) Hasher {
	if cost < 12 {
		cost = 12
	}
	if cost > bcrypt.MaxCost {
		cost = bcrypt.MaxCost
	}
	return Hasher{cost: cost}
}

// Hash returns the bcrypt hash of password. bcrypt silently truncates input at
// 72 bytes, so we reject longer inputs explicitly rather than surprise the user.
func (h Hasher) Hash(password string) (string, error) {
	if len(password) > 72 {
		return "", ErrPasswordTooLong
	}
	b, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Verify reports whether password matches hash (bcrypt compare is constant-time).
func (h Hasher) Verify(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
