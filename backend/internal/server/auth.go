package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/auth"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/db"
	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

type loginRequest struct {
	Email    string `json:"email"`
	Passwort string `json:"passwort"`
	Client   string `json:"client"` // "web" (default) | "mobile"
}

type arbeiterDTO struct {
	ID    uuid.UUID `json:"id"`
	Name  string    `json:"name"`
	Email string    `json:"email"`
	Rolle string    `json:"rolle"`
}

type tokenResponse struct {
	AccessToken  string      `json:"access_token"`
	RefreshToken string      `json:"refresh_token,omitempty"` // mobile only
	Arbeiter     arbeiterDTO `json:"arbeiter"`
}

func toArbeiterDTO(a db.Arbeiter) arbeiterDTO {
	return arbeiterDTO{ID: a.ID, Name: a.Name, Email: a.Email, Rolle: a.Rolle}
}

// handleLogin authenticates by email+password and issues tokens.
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültige Anfrage.")
		return
	}

	a, err := s.queries.GetArbeiterByEmail(r.Context(), normalizeEmail(req.Email))
	if errors.Is(err, pgx.ErrNoRows) {
		// Timing-safe: spend a bcrypt compare even for unknown emails.
		s.hasher.Verify(s.dummyHash, req.Passwort)
		platform.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "E-Mail oder Passwort falsch.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if !a.Aktiv || !s.hasher.Verify(a.PasswortHash, req.Passwort) {
		platform.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "E-Mail oder Passwort falsch.")
		return
	}
	s.issueTokens(w, r, a, req.Client)
}

// handleRefresh rotates the refresh token: validate -> revoke old -> issue new.
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	raw, client := s.readRefreshToken(r)
	if raw == "" {
		platform.WriteError(w, http.StatusUnauthorized, "unauthorized", "Kein Refresh-Token.")
		return
	}
	rt, err := s.queries.GetRefreshTokenByHash(r.Context(), auth.HashRefreshToken(raw))
	if errors.Is(err, pgx.ErrNoRows) {
		platform.WriteError(w, http.StatusUnauthorized, "unauthorized", "Refresh-Token ungültig.")
		return
	}
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	if rt.RevokedAt != nil || rt.ExpiresAt.Before(time.Now()) {
		// Replay of a revoked/expired token => treat the whole family as compromised.
		_ = s.queries.RevokeAllRefreshTokensForArbeiter(r.Context(), rt.ArbeiterID)
		platform.WriteError(w, http.StatusUnauthorized, "unauthorized", "Refresh-Token ungültig.")
		return
	}
	if err := s.queries.RevokeRefreshToken(r.Context(), rt.ID); err != nil {
		s.serverError(w, r, err)
		return
	}
	a, err := s.queries.GetArbeiterByID(r.Context(), rt.ArbeiterID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	s.issueTokens(w, r, a, client)
}

// handleLogout revokes the presented refresh token and clears the web cookie.
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	if raw, _ := s.readRefreshToken(r); raw != "" {
		if rt, err := s.queries.GetRefreshTokenByHash(r.Context(), auth.HashRefreshToken(raw)); err == nil {
			_ = s.queries.RevokeRefreshToken(r.Context(), rt.ID)
		}
	}
	s.clearRefreshCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

// handleMe returns the authenticated arbeiter (requireAuth guarantees identity).
func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	ident, _ := identityFrom(r.Context())
	a, err := s.queries.GetArbeiterByID(r.Context(), ident.ArbeiterID)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	platform.WriteJSON(w, http.StatusOK, toArbeiterDTO(a))
}

// issueTokens mints an access token + a rotating refresh token and delivers the
// refresh token per client: web => httpOnly cookie, mobile => JSON body.
func (s *Server) issueTokens(w http.ResponseWriter, r *http.Request, a db.Arbeiter, client string) {
	access, err := s.issuer.IssueAccessToken(a.ID, a.Rolle)
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	rawRefresh, hash, err := auth.NewRefreshToken()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	id, err := uuid.NewV7()
	if err != nil {
		s.serverError(w, r, err)
		return
	}
	expires := time.Now().Add(s.cfg.RefreshTokenTTL)
	if _, err := s.queries.CreateRefreshToken(r.Context(), db.CreateRefreshTokenParams{
		ID:         id,
		ArbeiterID: a.ID,
		TokenHash:  hash,
		ExpiresAt:  expires,
	}); err != nil {
		s.serverError(w, r, err)
		return
	}

	resp := tokenResponse{AccessToken: access, Arbeiter: toArbeiterDTO(a)}
	if client == "mobile" {
		resp.RefreshToken = rawRefresh
	} else {
		s.setRefreshCookie(w, rawRefresh, expires)
	}
	platform.WriteJSON(w, http.StatusOK, resp)
}

// readRefreshToken pulls the token from the web cookie or the mobile JSON body.
func (s *Server) readRefreshToken(r *http.Request) (raw, client string) {
	if c, err := r.Cookie(refreshCookieName); err == nil && c.Value != "" {
		return c.Value, "web"
	}
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	return body.RefreshToken, "mobile"
}
