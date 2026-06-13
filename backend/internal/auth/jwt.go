package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const tokenIssuer = "prekaj-zeiterfassung"

// ErrInvalidToken is returned for any malformed, expired, or wrongly-signed token.
var ErrInvalidToken = errors.New("auth: invalid token")

// Claims are the access-token claims: standard registered claims plus the role.
type Claims struct {
	Rolle string `json:"rolle"`
	jwt.RegisteredClaims
}

// Identity is the authenticated principal extracted from a valid access token.
type Identity struct {
	ArbeiterID uuid.UUID
	Rolle      string
}

// TokenIssuer signs and verifies short-lived HS256 access tokens.
type TokenIssuer struct {
	secret    []byte
	accessTTL time.Duration
}

// NewTokenIssuer fails fast if the secret is too weak (belt-and-braces against an
// empty/short JWT_SECRET; config also marks it required).
func NewTokenIssuer(secret string, accessTTL time.Duration) (*TokenIssuer, error) {
	if len(secret) < 32 {
		return nil, errors.New("auth: JWT secret must be >= 32 bytes")
	}
	return &TokenIssuer{secret: []byte(secret), accessTTL: accessTTL}, nil
}

// IssueAccessToken mints a signed access token for the given arbeiter and role.
func (ti *TokenIssuer) IssueAccessToken(arbeiterID uuid.UUID, rolle string) (string, error) {
	now := time.Now()
	claims := Claims{
		Rolle: rolle,
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    tokenIssuer,
			Subject:   arbeiterID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(ti.accessTTL)),
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(ti.secret)
}

// Verify parses and validates an access token, returning the principal.
func (ti *TokenIssuer) Verify(tokenString string) (Identity, error) {
	var claims Claims
	_, err := jwt.ParseWithClaims(tokenString, &claims,
		func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return ti.secret, nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
		jwt.WithIssuer(tokenIssuer),
		jwt.WithExpirationRequired(),
	)
	if err != nil {
		return Identity{}, errors.Join(ErrInvalidToken, err)
	}
	id, err := uuid.Parse(claims.Subject)
	if err != nil {
		return Identity{}, ErrInvalidToken
	}
	return Identity{ArbeiterID: id, Rolle: claims.Rolle}, nil
}
