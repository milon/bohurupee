package server

import (
	"encoding/json"
	"net/http"

	"github.com/milon/bohurupee/internal/config"
	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/oidc"
	"github.com/milon/bohurupee/internal/profiles"
)

// ApplyConfig swaps personas, PKCE, idToken mode, profiles, and refresh-token
// mode. Listen address and signing keys are unchanged. Invalid config leaves
// the previous values in place.
func (s *Server) ApplyConfig(cfg config.Config) error {
	personas := cfg.Personas
	if len(personas) == 0 {
		personas = []oauth.Persona{oauth.Alice}
	}
	catalog, err := oauth.NewCatalog(personas)
	if err != nil {
		return err
	}
	pkce := cfg.PKCE
	if pkce == "" {
		pkce = oauth.PKCEOptional
	}
	idToken := cfg.IDToken
	if idToken == "" {
		idToken = oidc.IDTokenOpenID
	}
	reg, err := profiles.NewRegistry(cfg.Profiles)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	s.catalog = catalog
	s.pkce = pkce
	s.idToken = idToken
	s.profiles = reg
	s.refreshTokens = cfg.RefreshTokens
	s.store.SetRefreshEnabled(cfg.RefreshTokens)
	return nil
}

// ReloadFromDisk reloads YAML from the path passed at startup (or
// ./bohurupee.yaml when that was the implicit default).
func (s *Server) ReloadFromDisk() error {
	s.mu.RLock()
	path := s.configPath
	s.mu.RUnlock()
	cfg, err := config.LoadPath(path)
	if err != nil {
		return err
	}
	return s.ApplyConfig(cfg)
}

func (s *Server) handleReload(w http.ResponseWriter, r *http.Request) {
	if !remoteIsLoopback(r) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	if err := s.ReloadFromDisk(); err != nil {
		writeTokenError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return
	}
	s.mu.RLock()
	personas := s.catalog.All()
	refresh := s.refreshTokens
	path := s.configPath
	s.mu.RUnlock()
	ids := make([]string, 0, len(personas))
	for _, p := range personas {
		ids = append(ids, p.ID)
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":            true,
		"config":        path,
		"personas":      ids,
		"refreshTokens": refresh,
	})
}

func (s *Server) snapshot() (catalog *oauth.Catalog, pkce oauth.PKCEMode, idToken oidc.IDTokenMode, reg *profiles.Registry, refresh bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.catalog, s.pkce, s.idToken, s.profiles, s.refreshTokens
}
