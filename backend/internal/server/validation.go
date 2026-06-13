package server

import (
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"github.com/go-playground/validator/v10"

	"github.com/lindritP/prekaj-zeiterfassung/backend/internal/platform"
)

// decodeAndValidate decodes the JSON body into dst and runs struct validation.
// On any failure it writes a 400 (central envelope) and returns false; the caller
// then simply returns. DisallowUnknownFields surfaces typo'd keys immediately.
func (s *Server) decodeAndValidate(w http.ResponseWriter, r *http.Request, dst any) bool {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		platform.WriteError(w, http.StatusBadRequest, "bad_request", "Ungültige Anfrage (JSON).")
		return false
	}
	if err := s.validate.Struct(dst); err != nil {
		var ve validator.ValidationErrors
		if errors.As(err, &ve) {
			platform.WriteError(w, http.StatusBadRequest, "validation_error", validationMessage(ve))
			return false
		}
		s.serverError(w, r, err) // non-ValidationErrors => programmer error
		return false
	}
	return true
}

// validationMessage renders a compact, German, field-aware message.
func validationMessage(ve validator.ValidationErrors) string {
	parts := make([]string, 0, len(ve))
	for _, fe := range ve {
		parts = append(parts, fieldRule(fe.Field(), fe.Tag(), fe.Param()))
	}
	return "Validierungsfehler: " + strings.Join(parts, "; ")
}

func fieldRule(field, tag, param string) string {
	switch tag {
	case "required":
		return field + " ist erforderlich"
	case "email":
		return field + " muss eine gültige E-Mail sein"
	case "min":
		return field + " ist zu kurz (min " + param + ")"
	case "max":
		return field + " ist zu lang (max " + param + ")"
	case "numeric":
		return field + " muss numerisch sein"
	case "oneof":
		return field + " muss einer von [" + param + "] sein"
	default:
		return field + " ist ungültig (" + tag + ")"
	}
}
