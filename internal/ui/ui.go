package ui

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io"
	"net/url"
	"strings"
	"unicode"

	"github.com/milon/bohurupee/assets"
	"github.com/milon/bohurupee/internal/oauth"
)

//go:embed *.html style.css
var files embed.FS

type Templates struct {
	home     *template.Template
	consent  *template.Template
	formPost *template.Template
}

type HomeData struct {
	Listen       string
	URL          string
	Version      string
	Personas     []oauth.Persona
	AuthorizeURL string
	Curl         string
	DiscoveryURL string
	ReloadURL    string
}

type ConsentData struct {
	Provider string
	Query    url.Values
	Personas []oauth.Persona
	LastID   string
}

type FormPostData struct {
	Action           string
	Code             string
	State            string
	Error            string
	ErrorDescription string
}

func denyURL(q url.Values) string {
	cp := make(url.Values, len(q)+1)
	for k, vs := range q {
		if k == "auto" {
			continue
		}
		cp[k] = append([]string{}, vs...)
	}
	cp.Set("deny", "1")
	return "?" + cp.Encode()
}

func Load() (*Templates, error) {
	css, err := files.ReadFile("style.css")
	if err != nil {
		return nil, fmt.Errorf("read style: %w", err)
	}
	funcMap := template.FuncMap{
		"continueURL": continueURL,
		"denyURL":     denyURL,
		"logo": func() template.HTML {
			return template.HTML(assets.Logo)
		},
		"css": func() template.CSS {
			return template.CSS(css)
		},
		"providerLabel": providerLabel,
	}
	home, err := template.New("home.html").Funcs(funcMap).ParseFS(files, "home.html")
	if err != nil {
		return nil, fmt.Errorf("parse home: %w", err)
	}
	consent, err := template.New("consent.html").Funcs(funcMap).ParseFS(files, "consent.html")
	if err != nil {
		return nil, fmt.Errorf("parse consent: %w", err)
	}
	formPost, err := template.ParseFS(files, "formpost.html")
	if err != nil {
		return nil, fmt.Errorf("parse formpost: %w", err)
	}
	return &Templates{home: home, consent: consent, formPost: formPost}, nil
}

func (t *Templates) WriteHome(w io.Writer, data HomeData) error {
	return t.home.Execute(w, data)
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

func providerLabel(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	if name, ok := knownProviders[slug]; ok {
		return name
	}
	parts := strings.FieldsFunc(slug, func(r rune) bool {
		return r == '-' || r == '_'
	})
	for i, part := range parts {
		runes := []rune(part)
		if len(runes) == 0 {
			continue
		}
		runes[0] = unicode.ToUpper(runes[0])
		parts[i] = string(runes)
	}
	return strings.Join(parts, " ")
}

var knownProviders = map[string]string{
	"google":    "Google",
	"github":    "GitHub",
	"gitlab":    "GitLab",
	"apple":     "Apple",
	"facebook":  "Facebook",
	"microsoft": "Microsoft",
	"linkedin":  "LinkedIn",
	"twitter":   "Twitter",
	"slack":     "Slack",
	"discord":   "Discord",
	"bitbucket": "Bitbucket",
	"jumpcloud": "JumpCloud",
	"okta":      "Okta",
	"auth0":     "Auth0",
}

func continueURL(q url.Values, personaID string) string {
	cp := make(url.Values, len(q)+1)
	for k, vs := range q {
		if k == "deny" {
			continue
		}
		cp[k] = append([]string{}, vs...)
	}
	cp.Set("auto", personaID)
	return "?" + cp.Encode()
}
