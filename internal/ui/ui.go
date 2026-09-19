package ui

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/url"

	"github.com/milon/bohurupee/internal/oauth"
)

//go:embed *.html
var files embed.FS

type Templates struct {
	consent  *template.Template
	formPost *template.Template
}

type ConsentData struct {
	Provider string
	Query    url.Values
	Personas []oauth.Persona
	LastID   string
}

type FormPostData struct {
	Action string
	Code   string
	State  string
}

func Load() (*Templates, error) {
	funcMap := template.FuncMap{
		"continueURL": continueURL,
	}
	consent, err := template.New("consent.html").Funcs(funcMap).ParseFS(files, "consent.html")
	if err != nil {
		return nil, fmt.Errorf("parse consent: %w", err)
	}
	formPost, err := template.ParseFS(files, "formpost.html")
	if err != nil {
		return nil, fmt.Errorf("parse formpost: %w", err)
	}
	return &Templates{consent: consent, formPost: formPost}, nil
}

func (t *Templates) WriteConsent(w io.Writer, data ConsentData) error {
	return t.consent.Execute(w, data)
}

func (t *Templates) RenderFormPost(data FormPostData) ([]byte, error) {
	var buf bytes.Buffer
	if err := t.formPost.Execute(&buf, data); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func continueURL(q url.Values, personaID string) string {
	cp := make(url.Values, len(q)+1)
	for k, vs := range q {
		cp[k] = append([]string{}, vs...)
	}
	cp.Set("auto", personaID)
	return "?" + cp.Encode()
}
