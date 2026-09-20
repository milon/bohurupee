package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/oidc"
	"github.com/milon/bohurupee/internal/profiles"
	"github.com/milon/bohurupee/internal/ui"
)

const (
	personaCookie = "bohurupee_persona"
	autoCookie    = "bohurupee_auto"
)

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
	Scope               string
	Nonce               string
	Prompt              string
	LoginHint           string
	Deny                bool
}

type authError struct {
	Redirect bool
	Code     string
	Desc     string
}

func (e authError) Error() string {
	if e.Desc != "" {
		return e.Desc
	}
	return e.Code
}

func (s *Server) handleAuthorize(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	catalog, pkce, _, reg, _ := s.snapshot()
	req, err := parseAuthRequest(r, pkce)
	if err != nil {
		s.writeAuthorizeError(w, req, err)
		return
	}
	if err := s.checkClientRedirect(req.ClientID, req.RedirectURI); err != nil {
		s.writeAuthorizeError(w, req, authError{Code: "invalid_request", Desc: err.Error()})
		return
	}
	s.applyProfileDefaults(&req, reg)

	if req.Deny {
		s.finishAuthorize(w, req, "", "access_denied", "the user denied the request")
		return
	}

	persona, err := s.resolvePersona(r, req, catalog)
	if err != nil {
		s.writeAuthorizeError(w, req, err)
		return
	}
	if persona == nil {
		s.renderConsent(w, r, req, catalog)
		return
	}

	code, err := s.store.IssueCode(oauth.IssueCodeParams{
		Provider:    req.Provider,
		ClientID:    req.ClientID,
		RedirectURI: req.RedirectURI,
		PersonaID:   persona.ID,
		Challenge:   req.CodeChallenge,
		Method:      req.CodeChallengeMethod,
		Scope:       req.Scope,
		Nonce:       req.Nonce,
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
	s.finishAuthorize(w, req, code, "", "")
}

func parseAuthRequest(r *http.Request, pkce oauth.PKCEMode) (authRequest, error) {
	provider := r.PathValue("provider")
	if !oauth.ValidProvider(provider) {
		return authRequest{}, authError{Code: "invalid_request", Desc: "invalid provider"}
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
		Scope:               strings.TrimSpace(q.Get("scope")),
		Nonce:               q.Get("nonce"),
		Prompt:              strings.TrimSpace(q.Get("prompt")),
		LoginHint:           strings.TrimSpace(q.Get("login_hint")),
		Deny:                truthy(q.Get("deny")),
	}
	if req.ClientID == "" {
		return req, authError{Code: "invalid_request", Desc: "missing client_id"}
	}
	if err := validateRedirectURI(req.RedirectURI); err != nil {
		return req, authError{Code: "invalid_request", Desc: err.Error()}
	}
	if req.ResponseType != "code" {
		return req, authError{Redirect: true, Code: "unsupported_response_type", Desc: "response_type must be code"}
	}
	if req.State == "" {
		return req, authError{Redirect: true, Code: "invalid_request", Desc: "missing state"}
	}
	switch req.ResponseMode {
	case "", "query", "form_post":
	default:
		return req, authError{Redirect: true, Code: "invalid_request", Desc: "response_mode must be query or form_post"}
	}
	method, err := oauth.NormalizeChallengeMethod(req.CodeChallengeMethod, req.CodeChallenge)
	if err != nil {
		return req, authError{Redirect: true, Code: "invalid_request", Desc: err.Error()}
	}
	req.CodeChallengeMethod = method
	if err := oauth.CheckAuthorizePKCE(pkce, req.CodeChallenge); err != nil {
		return req, authError{Redirect: true, Code: "invalid_request", Desc: err.Error()}
	}
	return req, nil
}

func (s *Server) applyProfileDefaults(req *authRequest, reg *profiles.Registry) {
	if req.ResponseMode == "" {
		if p, ok := reg.Get(req.Provider); ok && p.Protocol.ResponseMode != "" {
			req.ResponseMode = p.Protocol.ResponseMode
		} else {
			req.ResponseMode = "query"
		}
	}
}

func (s *Server) resolvePersona(r *http.Request, req authRequest, catalog *oauth.Catalog) (*oauth.Persona, error) {
	if req.Auto != "" {
		p, ok := catalog.Lookup(req.Auto)
		if !ok {
			return nil, authError{Redirect: true, Code: "invalid_request", Desc: fmt.Sprintf("unknown persona %q", req.Auto)}
		}
		return &p, nil
	}
	if promptHasLogin(req.Prompt) {
		return nil, nil
	}
	if s.autoApprove {
		p := catalog.Default()
		return &p, nil
	}
	if c, err := r.Cookie(autoCookie); err == nil {
		id := strings.TrimSpace(c.Value)
		if id != "" {
			p, ok := catalog.Lookup(id)
			if !ok {
				return nil, authError{Redirect: true, Code: "invalid_request", Desc: fmt.Sprintf("unknown persona %q", id)}
			}
			return &p, nil
		}
	}
	return nil, nil
}

func (s *Server) renderConsent(w http.ResponseWriter, r *http.Request, req authRequest, catalog *oauth.Catalog) {
	last := ""
	if hint := strings.TrimSpace(req.LoginHint); hint != "" {
		if _, ok := catalog.Lookup(hint); ok {
			last = hint
		}
	} else if !promptHasLogin(req.Prompt) {
		if c, err := r.Cookie(personaCookie); err == nil {
			if _, ok := catalog.Lookup(c.Value); ok {
				last = c.Value
			}
		}
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := s.ui.WriteConsent(w, ui.ConsentData{
		Provider: req.Provider,
		Query:    r.URL.Query(),
		Personas: catalog.All(),
		LastID:   last,
	}); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func promptHasLogin(prompt string) bool {
	for _, p := range strings.Fields(strings.ToLower(prompt)) {
		if p == "login" || p == "select_account" {
			return true
		}
	}
	return false
}

func (s *Server) checkClientRedirect(clientID, redirectURI string) error {
	open, clients := s.clientPolicy()
	if c, ok := clients[clientID]; ok && len(c.RedirectURIs) > 0 {
		for _, u := range c.RedirectURIs {
			if u == redirectURI {
				return nil
			}
		}
		return fmt.Errorf("redirect_uri is not registered for this client")
	}
	if open {
		return nil
	}
	if _, ok := clients[clientID]; ok {
		return nil
	}
	return fmt.Errorf("unknown client_id")
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

	catalog, _, idToken, reg, refreshOn := s.snapshot()

	grantType := r.PostFormValue("grant_type")
	if grantType == "" {
		grantType = r.FormValue("grant_type")
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

	var ex oauth.ExchangeResult
	switch grantType {
	case "authorization_code":
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
		ex, err = s.store.ExchangeCode(provider, clientID, redirectURI, code, verifier)
		if err != nil {
			writeTokenError(w, http.StatusBadRequest, "invalid_grant", err.Error())
			return
		}
	case "refresh_token":
		if !refreshOn {
			writeTokenError(w, http.StatusBadRequest, "unsupported_grant_type", "grant_type must be authorization_code")
			return
		}
		rt := firstForm(r, "refresh_token")
		if rt == "" {
			writeTokenError(w, http.StatusBadRequest, "invalid_request", "missing refresh_token")
			return
		}
		ex, err = s.store.RefreshAccess(provider, clientID, rt)
		if err != nil {
			writeTokenError(w, http.StatusBadRequest, "invalid_grant", err.Error())
			return
		}
	default:
		msg := "grant_type must be authorization_code"
		if refreshOn {
			msg = "grant_type must be authorization_code or refresh_token"
		}
		writeTokenError(w, http.StatusBadRequest, "unsupported_grant_type", msg)
		return
	}

	resp := struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		ExpiresIn    int    `json:"expires_in"`
		RefreshToken string `json:"refresh_token,omitempty"`
		IDToken      string `json:"id_token,omitempty"`
		Scope        string `json:"scope,omitempty"`
	}{
		AccessToken:  ex.Access,
		TokenType:    "Bearer",
		ExpiresIn:    ex.ExpiresIn,
		RefreshToken: ex.Refresh,
		Scope:        ex.Scope,
	}
	if wantIDToken(provider, ex.Scope, idToken, reg) {
		persona, ok := catalog.Lookup(ex.PersonaID)
		if !ok {
			writeTokenError(w, http.StatusInternalServerError, "server_error", "unknown persona")
			return
		}
		idt, err := s.signer.IDToken(oidc.IDTokenInput{
			Issuer:      s.issuer(provider),
			Audience:    ex.ClientID,
			Nonce:       ex.Nonce,
			Now:         s.now(),
			TTL:         s.tokenTTL,
			Provider:    provider,
			Persona:     persona,
			AccessToken: ex.Access,
		})
		if err != nil {
			writeTokenError(w, http.StatusInternalServerError, "server_error", "could not issue id_token")
			return
		}
		resp.IDToken = idt
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	_ = json.NewEncoder(w).Encode(resp)
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
	catalog, _, _, reg, _ := s.snapshot()
	persona, ok := catalog.Lookup(personaID)
	if !ok {
		http.Error(w, "unknown persona", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	p, _ := reg.Get(provider)
	_ = json.NewEncoder(w).Encode(profiles.Render(provider, persona, p))
}

func wantIDToken(provider, scope string, mode oidc.IDTokenMode, reg *profiles.Registry) bool {
	if p, ok := reg.Get(provider); ok && p.Protocol.IDToken {
		return true
	}
	return oidc.WantIDToken(mode, scope)
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

func redirectWithQuery(redirectURI string, extra url.Values) (string, error) {
	u, err := url.Parse(redirectURI)
	if err != nil {
		return "", err
	}
	q := u.Query()
	for k, vs := range extra {
		for _, v := range vs {
			q.Set(k, v)
		}
	}
	u.RawQuery = q.Encode()
	return u.String(), nil
}

func (s *Server) writeAuthorizeError(w http.ResponseWriter, req authRequest, err error) {
	ae, ok := err.(authError)
	if !ok {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if ae.Redirect && validateRedirectURI(req.RedirectURI) == nil {
		s.finishAuthorize(w, req, "", ae.Code, ae.Desc)
		return
	}
	http.Error(w, ae.Error(), http.StatusBadRequest)
}

func (s *Server) finishAuthorize(w http.ResponseWriter, req authRequest, code, errCode, errDesc string) {
	_, _, _, reg, _ := s.snapshot()
	s.applyProfileDefaults(&req, reg)
	w.Header().Set("Cache-Control", "no-store")
	if req.ResponseMode == "form_post" {
		html, err := s.ui.RenderFormPost(ui.FormPostData{
			Action:           req.RedirectURI,
			Code:             code,
			State:            req.State,
			Error:            errCode,
			ErrorDescription: errDesc,
		})
		if err != nil {
			http.Error(w, "template error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		_, _ = w.Write(html)
		return
	}
	q := url.Values{}
	if errCode != "" {
		q.Set("error", errCode)
		if errDesc != "" {
			q.Set("error_description", errDesc)
		}
	} else {
		q.Set("code", code)
	}
	q.Set("state", req.State)
	loc, err := redirectWithQuery(req.RedirectURI, q)
	if err != nil {
		http.Error(w, "invalid redirect_uri", http.StatusBadRequest)
		return
	}
	w.Header().Set("Location", loc)
	w.WriteHeader(http.StatusFound)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if r.Method == http.MethodPost {
		if err := r.ParseForm(); err != nil {
			writeTokenError(w, http.StatusBadRequest, "invalid_request", "could not parse form")
			return
		}
	}
	personaID := strings.TrimSpace(firstForm(r, "persona"))
	if personaID == "" {
		personaID = strings.TrimSpace(r.URL.Query().Get("persona"))
	}
	provider := strings.TrimSpace(firstForm(r, "provider"))
	if provider == "" {
		provider = strings.TrimSpace(r.URL.Query().Get("provider"))
	}
	if provider != "" && !oauth.ValidProvider(provider) {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", "invalid provider")
		return
	}
	catalog, _, _, _, _ := s.snapshot()
	p, ok := catalog.Lookup(personaID)
	if !ok {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", fmt.Sprintf("unknown persona %q", personaID))
		return
	}
	http.SetCookie(w, &http.Cookie{
		Name:     autoCookie,
		Value:    p.ID,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		SameSite: http.SameSiteLaxMode,
		HttpOnly: true,
	})
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]string{
		"persona":  p.ID,
		"provider": provider,
	})
}

func truthy(v string) bool {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
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
