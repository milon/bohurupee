package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/milon/bohurupee/internal/oauth"
)

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if !oauth.ValidProvider(provider) {
		http.Error(w, "invalid provider", http.StatusBadRequest)
		return
	}

	q := r.URL.Query()
	clientID := strings.TrimSpace(q.Get("client_id"))
	redirectURI := strings.TrimSpace(q.Get("redirect_uri"))
	responseType := q.Get("response_type")
	state := q.Get("state")

	if clientID == "" {
		http.Error(w, "missing client_id", http.StatusBadRequest)
		return
	}
	if err := validateRedirectURI(redirectURI); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if responseType != "code" {
		http.Error(w, "response_type must be code", http.StatusBadRequest)
		return
	}
	if state == "" {
		http.Error(w, "missing state", http.StatusBadRequest)
		return
	}

	persona, err := s.autoPersona(q.Get("auto"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	code, err := s.store.IssueCode(provider, clientID, redirectURI, persona.ID)
	if err != nil {
		http.Error(w, "could not issue code", http.StatusInternalServerError)
		return
	}

	loc, err := redirectWithCode(redirectURI, code, state)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	http.Redirect(w, r, loc, http.StatusFound)
}

func (s *Server) autoPersona(auto string) (oauth.Persona, error) {
	auto = strings.TrimSpace(auto)
	if auto != "" {
		p, ok := oauth.PersonaByID(auto)
		if !ok {
			return oauth.Persona{}, fmt.Errorf("unknown persona %q (only alice is available)", auto)
		}
		return p, nil
	}
	if s.autoApprove {
		return oauth.Alice, nil
	}
	return oauth.Persona{}, fmt.Errorf("auto-approve required: pass ?auto=alice or set BOHURUPEE_AUTO_APPROVE=1")
}

func (s *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if !oauth.ValidProvider(provider) {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", "invalid provider")
		return
	}
	if err := r.ParseForm(); err != nil {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", "could not parse form")
		return
	}

	grantType := r.PostFormValue("grant_type")
	if grantType == "" {
		grantType = r.FormValue("grant_type")
	}
	if grantType != "authorization_code" {
		writeTokenError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code")
		return
	}

	clientID, _, err := clientCredentials(r)
	if err != nil {
		writeTokenError(w, http.StatusUnauthorized, "invalid_client", err.Error())
		return
	}
	if clientID == "" {
		writeTokenError(w, http.StatusUnauthorized, "invalid_client", "missing client_id")
		return
	}

	code := firstForm(r, "code")
	redirectURI := firstForm(r, "redirect_uri")
	if code == "" {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", "missing code")
		return
	}
	if redirectURI == "" {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", "missing redirect_uri")
		return
	}

	access, expiresIn, _, err := s.store.ExchangeCode(provider, clientID, redirectURI, code)
	if err != nil {
		writeTokenError(w, http.StatusBadRequest, "invalid_grant", err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_ = json.NewEncoder(w).Encode(struct {
		AccessToken string `json:"access_token"`
		TokenType   string `json:"token_type"`
		ExpiresIn   int    `json:"expires_in"`
	}{
		AccessToken: access,
		TokenType:   "Bearer",
		ExpiresIn:   expiresIn,
	})
}

func (s *Server) handleUserinfo(w http.ResponseWriter, r *http.Request) {
	provider := r.PathValue("provider")
	if !oauth.ValidProvider(provider) {
		http.Error(w, "invalid provider", http.StatusBadRequest)
		return
	}

	token := bearerToken(r.Header.Get("Authorization"))
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer`)
		http.Error(w, "missing bearer token", http.StatusUnauthorized)
		return
	}

	personaID, err := s.store.LookupToken(provider, token)
	if err != nil {
		w.Header().Set("WWW-Authenticate", `Bearer error="invalid_token"`)
		http.Error(w, err.Error(), http.StatusUnauthorized)
		return
	}
	persona, ok := oauth.PersonaByID(personaID)
	if !ok {
		http.Error(w, "unknown persona", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(persona.Userinfo(provider))
}

func clientCredentials(r *http.Request) (id, secret string, err error) {
	id = firstForm(r, "client_id")
	secret = firstForm(r, "client_secret")
	if user, pass, ok := r.BasicAuth(); ok {
		if id != "" && id != user {
			return "", "", fmt.Errorf("client_id mismatch between body and Authorization")
		}
		id = user
		if secret == "" {
			secret = pass
		}
	}
	return id, secret, nil
}

func firstForm(r *http.Request, key string) string {
	if v := r.PostFormValue(key); v != "" {
		return v
	}
	return r.FormValue(key)
}

func bearerToken(header string) string {
	const prefix = "bearer "
	if len(header) < len(prefix) {
		return ""
	}
	if !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

func validateRedirectURI(raw string) error {
	if raw == "" {
		return fmt.Errorf("missing redirect_uri")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return fmt.Errorf("invalid redirect_uri")
	}
	if u.Fragment != "" {
		return fmt.Errorf("redirect_uri must not include a fragment")
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("redirect_uri must be http or https")
	}
	if u.Host == "" {
		return fmt.Errorf("redirect_uri must be absolute")
	}
	return nil
}

func redirectWithCode(redirectURI, code, state string) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	q := u.Query()
	q.Set("code", code)
	q.Set("state", state)
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func writeTokenError(w http.ResponseWriter, status int, code, desc string) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(map[string]string{
		"error":             code,
		"error_description": desc,
	})
}
