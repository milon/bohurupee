package server

import (
	"fmt"
	"html/template"
	"net/http"

	"github.com/milon/bohurupee/internal/listen"
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
</body>
</html>
`

type Server struct {
	Addr listen.Addr
	mux  *http.ServeMux
	page *template.Template
}

func New(addr listen.Addr) (*Server, error) {
	t, err := template.New("home").Parse(homeTmpl)
	if err != nil {
		return nil, fmt.Errorf("parse home template: %w", err)
	}
	s := &Server{
		Addr: addr,
		mux:  http.NewServeMux(),
		page: t,
	}
	s.mux.HandleFunc("GET /{$}", s.handleHome)
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
