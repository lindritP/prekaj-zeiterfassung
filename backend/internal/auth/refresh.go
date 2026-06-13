package auth

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
)

// NewRefreshToken returns an opaque, high-entropy token (URL-safe, ~256 bits) and
// its sha256 hash. Only the hash is stored; the raw token goes to the client.
//
// Why sha256 and not bcrypt: bcrypt protects LOW-entropy human passwords by being
// deliberately slow. A refresh token is 32 random bytes (2^256 space), so brute
// force is already infeasible — a fast cryptographic hash is the right tool, and a
// bytea UNIQUE index on the hash gives O(1) lookups (bcrypt's per-row salt can't be
// indexed by value). We still hash so a DB leak can't replay tokens.
func NewRefreshToken() (raw string, hash []byte, err error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", nil, err
	}
	raw = base64.RawURLEncoding.EncodeToString(b)
	return raw, HashRefreshToken(raw), nil
}

// HashRefreshToken returns the sha256 of the raw token for storage/lookup.
func HashRefreshToken(raw string) []byte {
	sum := sha256.Sum256([]byte(raw))
	return sum[:]
}
