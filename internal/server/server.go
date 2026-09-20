package server

import (
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/milon/bohurupee/assets"
	"github.com/milon/bohurupee/internal/config"
	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/oidc"
	"github.com/milon/bohurupee/internal/profiles"
	"github.com/milon/bohurupee/internal/ui"
	"github.com/milon/bohurupee/internal/version"
)

type Options struct {
	Addr          listen.Addr
	AutoApprove   bool
	Clock         oauth.Clock
	CodeTTL       time.Duration
	TokenTTL      time.Duration
	Personas      []oauth.Persona
	PKCE          oauth.PKCEMode
	Signer        *oidc.Signer
	IDToken       oidc.IDTokenMode
	Profiles      map[string]profiles.Profile
	ConfigPath    string
	RefreshTokens bool
	// OpenClient defaults to true when nil (zero Options in tests).
	OpenClient *bool
	Clients    map[string]config.Client
}

type Server struct {
	Addr          listen.Addr
	mux           *http.ServeMux
	ui            *ui.Templates
	store         *oauth.Store
	mu            sync.RWMutex
	catalog       *oauth.Catalog
	autoApprove   bool
	pkce          oauth.PKCEMode
	clock         oauth.Clock
	tokenTTL      time.Duration
	signer        *oidc.Signer
	idToken       oidc.IDTokenMode
	profiles      *profiles.Registry
	configPath    string
	refreshTokens bool
	openClient    bool
	clients       map[string]config.Client
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
	store := oauth.NewStore(opts.Clock, opts.CodeTTL, tokenTTL)
	store.SetRefreshEnabled(opts.RefreshTokens)
	openClient := true
	if opts.OpenClient != nil {
		openClient = *opts.OpenClient
	}
	s := &Server{
		Addr:          opts.Addr,
		mux:           http.NewServeMux(),
		ui:            pages,
		store:         store,
		catalog:       catalog,
		autoApprove:   opts.AutoApprove,
		pkce:          pkce,
		clock:         opts.Clock,
		tokenTTL:      tokenTTL,
		signer:        signer,
		idToken:       idToken,
		profiles:      reg,
		configPath:    opts.ConfigPath,
		refreshTokens: opts.RefreshTokens,
		openClient:    openClient,
		clients:       opts.Clients,
	}
	s.mux.HandleFunc("GET /{$}", s.handleHome)
	s.mux.HandleFunc("GET /favicon.svg", s.handleFavicon)
	s.mux.HandleFunc("GET /__login", s.handleLogin)
	s.mux.HandleFunc("POST /__login", s.handleLogin)
	s.mux.HandleFunc("POST /__reload", s.handleReload)
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
	return corsLoopback(s.mux)
}

func (s *Server) handleHome(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	base := s.Addr.DisplayURL()
	authorize, curlCmd := homeSnippets(base)
	catalog, _, _, _, _ := s.snapshot()
	data := ui.HomeData{
		Listen:       s.Addr.String(),
		URL:          base,
		Version:      version.String(),
		Personas:     catalog.All(),
		AuthorizeURL: authorize,
		Curl:         curlCmd,
		DiscoveryURL: base + "/google/.well-known/openid-configuration",
		ReloadURL:    base + "/__reload",
	}
	if err := s.ui.WriteHome(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}

func homeSnippets(base string) (authorize, curlCmd string) {
	q := url.Values{}
	q.Set("client_id", "dev-client")
	q.Set("redirect_uri", "http://127.0.0.1:9999/callback")
	q.Set("response_type", "code")
	q.Set("state", "xyz")
	authorize = base + "/google/authorize?" + q.Encode()
	q.Set("auto", "alice")
	curlCmd = "curl -sI '" + base + "/google/authorize?" + q.Encode() + "'"
	return authorize, curlCmd
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
