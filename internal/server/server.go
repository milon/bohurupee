package server

import (
	"net/http"
	"time"

	"github.com/milon/bohurupee/assets"
	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/oidc"
	"github.com/milon/bohurupee/internal/profiles"
	"github.com/milon/bohurupee/internal/ui"
)

type Options struct {
	Addr        listen.Addr
	AutoApprove bool
	Clock       oauth.Clock
	CodeTTL     time.Duration
	TokenTTL    time.Duration
	Personas    []oauth.Persona
	PKCE        oauth.PKCEMode
	Signer      *oidc.Signer
	IDToken     oidc.IDTokenMode
	Profiles    map[string]profiles.Profile
}

type Server struct {
	Addr        listen.Addr
	mux         *http.ServeMux
	ui          *ui.Templates
	store       *oauth.Store
	catalog     *oauth.Catalog
	autoApprove bool
	pkce        oauth.PKCEMode
	clock       oauth.Clock
	tokenTTL    time.Duration
	signer      *oidc.Signer
	idToken     oidc.IDTokenMode
	profiles    *profiles.Registry
}

func New(addr listen.Addr) (*Server, error) {
	return NewWithOptions(Options{Addr: addr})
}

func NewWithOptions(opts Options) (*Server, error) {
	pages, err := ui.Load()
	if err != nil {
		return nil, err
	}
	personas := opts.Personas
	if len(personas) == 0 {
		personas = []oauth.Persona{oauth.Alice}
	}
	catalog, err := oauth.NewCatalog(personas)
	if err != nil {
		return nil, err
	}
	pkce := opts.PKCE
	if pkce == "" {
		pkce = oauth.PKCEOptional
	}
	signer := opts.Signer
	if signer == nil {
		var err error
		signer, err = oidc.Generate()
		if err != nil {
			return nil, err
		}
	}
	idToken := opts.IDToken
	if idToken == "" {
		idToken = oidc.IDTokenOpenID
	}
	tokenTTL := opts.TokenTTL
	if tokenTTL <= 0 {
		tokenTTL = oauth.DefaultTokenTTL
	}
	reg, err := profiles.NewRegistry(opts.Profiles)
	if err != nil {
		return nil, err
	}
	s := &Server{
		Addr:        opts.Addr,
		mux:         http.NewServeMux(),
		ui:          pages,
		store:       oauth.NewStore(opts.Clock, opts.CodeTTL, tokenTTL),
		catalog:     catalog,
		autoApprove: opts.AutoApprove,
		pkce:        pkce,
		clock:       opts.Clock,
		tokenTTL:    tokenTTL,
		signer:      signer,
		idToken:     idToken,
		profiles:    reg,
	}
	s.mux.HandleFunc("GET /{$}", s.handleHome)
	s.mux.HandleFunc("GET /favicon.svg", s.handleFavicon)
	s.mux.HandleFunc("GET /{provider}/authorize", s.handleAuthorize)
	s.mux.HandleFunc("POST /{provider}/token", s.handleToken)
	s.mux.HandleFunc("GET /{provider}/userinfo", s.handleUserinfo)
	s.mux.HandleFunc("GET /{provider}/.well-known/openid-configuration", s.handleDiscovery)
	s.mux.HandleFunc("GET /{provider}/jwks", s.handleJWKS)
	s.mux.HandleFunc("GET /{provider}/auth/keys", s.handleJWKS)
	for _, alias := range reg.UserinfoAliases() {
		s.mux.HandleFunc("GET /{provider}/"+alias, s.handleUserinfo)
	}
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHome(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	data := ui.HomeData{
		Listen: s.Addr.String(),
		URL:    s.Addr.DisplayURL(),
	}
	if err := s.ui.WriteHome(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func (s *Server) handleFavicon(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "public, max-age=86400")
	_, _ = w.Write(assets.Favicon)
}

func (s *Server) now() time.Time {
	if s.clock != nil {
		return s.clock.Now()
	}
	return time.Now()
}
