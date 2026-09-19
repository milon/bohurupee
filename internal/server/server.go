package server

import (
	"fmt"
	"html/template"
	"net/http"
	"time"

	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/ui"
)

const homeTmpl = `<!DOCTYPE html>
<html lang="en">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>Bohurupee — DEV ONLY</title>
  <style>
    body { font-family: ui-sans-serif, system-ui, sans-serif; margin: 2.5rem auto; max-width: 40rem; line-height: 1.5; color: #111; }
    .banner { background: #b45309; color: #fff; font-weight: 700; letter-spacing: 0.04em; padding: 0.4rem 0.75rem; display: inline-block; }
    code { background: #f3f4f6; padding: 0.1rem 0.35rem; }
  </style>
</head>
<body>
  <p class="banner">DEV ONLY</p>
  <h1>Bohurupee — DEV ONLY</h1>
  <p>Local fake identity provider. Do not expose this process beyond your machine.</p>
  <p>Listening at <code>{{.Listen}}</code></p>
  <p>Open <a href="{{.URL}}">{{.URL}}</a></p>
  <p>OAuth: <code>/{provider}/authorize</code> (consent), <code>/{provider}/token</code>, <code>/{provider}/userinfo</code></p>
</body>
</html>
`

type Options struct {
	Addr        listen.Addr
	AutoApprove bool
	Clock       oauth.Clock
	CodeTTL     time.Duration
	TokenTTL    time.Duration
	Personas    []oauth.Persona
	PKCE        oauth.PKCEMode
}

type Server struct {
	Addr        listen.Addr
	mux         *http.ServeMux
	page        *template.Template
	ui          *ui.Templates
	store       *oauth.Store
	catalog     *oauth.Catalog
	autoApprove bool
	pkce        oauth.PKCEMode
}

func New(addr listen.Addr) (*Server, error) {
	return NewWithOptions(Options{Addr: addr})
}

func NewWithOptions(opts Options) (*Server, error) {
	t, err := template.New("home").Parse(homeTmpl)
	if err != nil {
		return nil, fmt.Errorf("parse home template: %w", err)
	}
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
	s := &Server{
		Addr:        opts.Addr,
		mux:         http.NewServeMux(),
		page:        t,
		ui:          pages,
		store:       oauth.NewStore(opts.Clock, opts.CodeTTL, opts.TokenTTL),
		catalog:     catalog,
		autoApprove: opts.AutoApprove,
		pkce:        pkce,
	}
	s.mux.HandleFunc("GET /{$}", s.handleHome)
	s.mux.HandleFunc("GET /{provider}/authorize", s.handleAuthorize)
	s.mux.HandleFunc("POST /{provider}/token", s.handleToken)
	s.mux.HandleFunc("GET /{provider}/userinfo", s.handleUserinfo)
	return s, nil
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleHome(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	data := struct {
		Listen string
		URL    string
	}{
		Listen: s.Addr.String(),
		URL:    s.Addr.DisplayURL(),
	}
	if err := s.page.Execute(w, data); err != nil {
		http.Error(w, "template error", http.StatusInternalServerError)
	}
}
