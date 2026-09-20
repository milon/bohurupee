package oauth

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync"
	"time"
)

const (
	DefaultCodeTTL    = 2 * time.Minute
	DefaultTokenTTL   = time.Hour
	DefaultRefreshTTL = 24 * time.Hour
)

type Store struct {
	mu             sync.Mutex
	codes          map[string]*codeGrant
	tokens         map[string]*accessToken
	refresh        map[string]*refreshGrant
	clock          Clock
	codeTTL        time.Duration
	tokenTTL       time.Duration
	refreshTTL     time.Duration
	refreshEnabled bool
}

type codeGrant struct {
	Provider    string
	ClientID    string
	RedirectURI string
	PersonaID   string
	Challenge   string
	Method      string
	Scope       string
	Nonce       string
	ExpiresAt   time.Time
	Used        bool
}

type accessToken struct {
	Provider  string
	PersonaID string
	ExpiresAt time.Time
}

type refreshGrant struct {
	Provider  string
	PersonaID string
	ClientID  string
	Scope     string
	Nonce     string
	ExpiresAt time.Time
}

func NewStore(clock Clock, codeTTL, tokenTTL time.Duration) *Store {
	if clock == nil {
		clock = realClock{}
	}
	if codeTTL <= 0 {
		codeTTL = DefaultCodeTTL
	}
	if tokenTTL <= 0 {
		tokenTTL = DefaultTokenTTL
	}
	return &Store{
		codes:      make(map[string]*codeGrant),
		tokens:     make(map[string]*accessToken),
		refresh:    make(map[string]*refreshGrant),
		clock:      clock,
		codeTTL:    codeTTL,
		tokenTTL:   tokenTTL,
		refreshTTL: DefaultRefreshTTL,
	}
}

// SetRefreshEnabled toggles whether authorization_code exchanges issue a
// refresh_token. Existing refresh grants stay usable until they expire.
func (s *Store) SetRefreshEnabled(v bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.refreshEnabled = v
}

type IssueCodeParams struct {
	Provider    string
	ClientID    string
	RedirectURI string
	PersonaID   string
	Challenge   string
	Method      string
	Scope       string
	Nonce       string
}

func (s *Store) IssueCode(p IssueCodeParams) (string, error) {
	code, err := randomToken()
	if err != nil {
		return "", err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.codes[code] = &codeGrant{
		Provider:    p.Provider,
		ClientID:    p.ClientID,
		RedirectURI: p.RedirectURI,
		PersonaID:   p.PersonaID,
		Challenge:   p.Challenge,
		Method:      p.Method,
		Scope:       p.Scope,
		Nonce:       p.Nonce,
		ExpiresAt:   s.clock.Now().Add(s.codeTTL),
	}
	return code, nil
}

type ExchangeResult struct {
	Access    string
	Refresh   string
	ExpiresIn int
	PersonaID string
	ClientID  string
	Provider  string
	Scope     string
	Nonce     string
}

func (s *Store) ExchangeCode(provider, clientID, redirectURI, code, verifier string) (ExchangeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	grant, ok := s.codes[code]
	if !ok {
		return ExchangeResult{}, fmt.Errorf("unknown code")
	}
	now := s.clock.Now()
	if grant.Used {
		return ExchangeResult{}, fmt.Errorf("code already used")
	}
	if !now.Before(grant.ExpiresAt) {
		delete(s.codes, code)
		return ExchangeResult{}, fmt.Errorf("code expired")
	}
	if grant.Provider != provider {
		return ExchangeResult{}, fmt.Errorf("code issued for a different provider")
	}
	if grant.ClientID != clientID {
		return ExchangeResult{}, fmt.Errorf("client_id mismatch")
	}
	if grant.RedirectURI != redirectURI {
		return ExchangeResult{}, fmt.Errorf("redirect_uri mismatch")
	}
	if err := VerifyPKCE(grant.Method, grant.Challenge, verifier); err != nil {
		return ExchangeResult{}, err
	}

	grant.Used = true
	delete(s.codes, code)

	return s.issueTokensLocked(now, provider, grant.PersonaID, grant.ClientID, grant.Scope, grant.Nonce, s.refreshEnabled)
}

// RefreshAccess exchanges a refresh_token for a new access token.
func (s *Store) RefreshAccess(provider, clientID, refreshToken string) (ExchangeResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	grant, ok := s.refresh[refreshToken]
	if !ok {
		return ExchangeResult{}, fmt.Errorf("unknown refresh_token")
	}
	now := s.clock.Now()
	if !now.Before(grant.ExpiresAt) {
		delete(s.refresh, refreshToken)
		return ExchangeResult{}, fmt.Errorf("refresh_token expired")
	}
	if grant.Provider != provider {
		return ExchangeResult{}, fmt.Errorf("refresh_token issued for a different provider")
	}
	if grant.ClientID != clientID {
		return ExchangeResult{}, fmt.Errorf("client_id mismatch")
	}

	ex, err := s.issueTokensLocked(now, provider, grant.PersonaID, grant.ClientID, grant.Scope, grant.Nonce, false)
	if err != nil {
		return ExchangeResult{}, err
	}
	ex.Refresh = refreshToken
	return ex, nil
}

func (s *Store) issueTokensLocked(now time.Time, provider, personaID, clientID, scope, nonce string, withRefresh bool) (ExchangeResult, error) {
	token, err := randomToken()
	if err != nil {
		return ExchangeResult{}, err
	}
	ttl := s.tokenTTL
	s.tokens[token] = &accessToken{
		Provider:  provider,
		PersonaID: personaID,
		ExpiresAt: now.Add(ttl),
	}
	ex := ExchangeResult{
		Access:    token,
		ExpiresIn: int(ttl / time.Second),
		PersonaID: personaID,
		ClientID:  clientID,
		Provider:  provider,
		Scope:     scope,
		Nonce:     nonce,
	}
	if withRefresh {
		rt, err := randomToken()
		if err != nil {
			return ExchangeResult{}, err
		}
		s.refresh[rt] = &refreshGrant{
			Provider:  provider,
			PersonaID: personaID,
			ClientID:  clientID,
			Scope:     scope,
			Nonce:     nonce,
			ExpiresAt: now.Add(s.refreshTTL),
		}
		ex.Refresh = rt
	}
	return ex, nil
}

func (s *Store) LookupToken(provider, token string) (personaID string, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	at, ok := s.tokens[token]
	if !ok {
		return "", fmt.Errorf("unknown token")
	}
	if !s.clock.Now().Before(at.ExpiresAt) {
		delete(s.tokens, token)
		return "", fmt.Errorf("token expired")
	}
	if at.Provider != provider {
		return "", fmt.Errorf("token issued for a different provider")
	}
	return at.PersonaID, nil
}

func randomToken() (string, error) {
	var b [32]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(b[:]), nil
}
