package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/ui"
)

const personaCookie = "bohurupee_persona"

type authRequest struct {
	Provider            string
	ClientID            string
	RedirectURI         string
	ResponseType        string
	State               string
	ResponseMode        string
	CodeChallenge       string
	CodeChallengeMethod string
	Auto                string
}

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	req, err := parseAuthRequest(r, s.pkce)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	persona, err := s.resolvePersona(req.Auto)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if persona == nil {
		s.renderConsent(w, r, req)
		return
	}

	code, err := s.store.IssueCode(oauth.IssueCodeParams{
		Provider:    req.Provider,
		ClientID:    req.ClientID,
		RedirectURI: req.RedirectURI,
		PersonaID:   persona.ID,
		Challenge:   req.CodeChallenge,
		Method:      req.CodeChallengeMethod,
	})
	if err != nil {
		http.Error(w, "could not issue code", http.StatusInternalServerError)
		return
	}

	http.SetCookie(w, &http.Cookie{
		Name:     personaCookie,
		Value:    persona.ID,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	w.Header().Set("Cache-Control", "no-store")

	if req.ResponseMode == "form_post" {
		html, err := s.ui.RenderFormPost(ui.FormPostData{
			Action: req.RedirectURI,
			Code:   code,
			State:  req.State,
		})
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
		return
	}

	loc, err := redirectWithCode(req.RedirectURI, code, req.State)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, loc, http.StatusFound)
}

func parseAuthRequest(r *http.Request, pkce oauth.PKCEMode) (authRequest, error) {
	provider := r.PathValue("provider")
	if !oauth.ValidProvider(provider) {
		return authRequest{}, fmt.Errorf("invalid provider")
	}
	q := r.URL.Query()
	req := authRequest{
		Provider:            provider,
		ClientID:            strings.TrimSpace(q.Get("client_id")),
		RedirectURI:         strings.TrimSpace(q.Get("redirect_uri")),
		ResponseType:        q.Get("response_type"),
		State:               q.Get("state"),
		ResponseMode:        q.Get("response_mode"),
		CodeChallenge:       q.Get("code_challenge"),
		CodeChallengeMethod: q.Get("code_challenge_method"),
		Auto:                strings.TrimSpace(q.Get("auto")),
	}
	if req.ClientID == "" {
		return authRequest{}, fmt.Errorf("missing client_id")
	}
	if err := validateRedirectURI(req.RedirectURI); err != nil {
		return authRequest{}, err
	}
	if req.ResponseType != "code" {
		return authRequest{}, fmt.Errorf("response_type must be code")
	}
	if req.State == "" {
		return authRequest{}, fmt.Errorf("missing state")
	}
	switch req.ResponseMode {
	case "", "query":
		req.ResponseMode = "query"
	case "form_post":
	default:
		return authRequest{}, fmt.Errorf("response_mode must be query or form_post")
	}
	method, err := oauth.NormalizeChallengeMethod(req.CodeChallengeMethod, req.CodeChallenge)
	if err != nil {
		return authRequest{}, err
	}
	req.CodeChallengeMethod = method
	if err := oauth.CheckAuthorizePKCE(pkce, req.CodeChallenge); err != nil {
		return authRequest{}, err
	}
	return req, nil
}

func (s *Server) resolvePersona(auto string) (*oauth.Persona, error) {
	if auto != "" {
		p, ok := s.catalog.Lookup(auto)
		if !ok {
			return nil, fmt.Errorf("unknown persona %q", auto)
		}
		return &p, nil
	}
	if s.autoApprove {
		p := s.catalog.Default()
		return &p, nil
	}
	return nil, nil
}

func (s *Server) renderConsent(w http.ResponseWriter, r *http.Request, req authRequest) {
	last := ""
	if c, err := r.Cookie(personaCookie); err == nil {
		if _, ok := s.catalog.Lookup(c.Value); ok {
			last = c.Value
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.ui.WriteConsent(w, ui.ConsentData{
		Provider: req.Provider,
		Query:    r.URL.Query(),
		Personas: s.catalog.All(),
		LastID:   last,
	}); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
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
	verifier := firstForm(r, "code_verifier")
	if code == "" {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", "missing code")
		return
	}
	if redirectURI == "" {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", "missing redirect_uri")
		return
	}

	access, expiresIn, _, err := s.store.ExchangeCode(provider, clientID, redirectURI, code, verifier)
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
	persona, ok := s.catalog.Lookup(personaID)
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
