package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/milon/bohurupee/internal/listen"
	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/profiles"
)

func TestAcmeUserinfoStaysGeneric(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
		Profiles: map[string]profiles.Profile{
			"github": {Template: "github"},
		},
	})
	info := completeFlow(t, srv, "acme")
	if info.ID != "acme:alice" || info.Nickname != "alice" {
		t.Fatalf("%+v", info)
	}
	raw := userinfoMap(t, srv, "acme", authorizeCode(t, srv, "acme", false))
	if _, ok := raw["login"]; ok {
		t.Fatalf("generic acme should not have login: %v", raw)
	}
}

func TestGitHubProfileShapesUserinfo(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
		Profiles: map[string]profiles.Profile{
			"github": {Template: "github"},
		},
	})
	raw := userinfoMap(t, srv, "github", authorizeCode(t, srv, "github", false))
	if raw["login"] != "alice" || raw["avatar_url"] != oauth.Alice.Avatar {
		t.Fatalf("%v", raw)
	}
	if raw["id"] != "github:alice" || raw["email"] != oauth.Alice.Email || raw["name"] != oauth.Alice.Name {
		t.Fatalf("getters missing: %v", raw)
	}
}

func TestCustomResponseWithoutTemplate(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
		Profiles: map[string]profiles.Profile{
			"acme": {Response: map[string]any{"badge": "ok", "handle": "{{nickname}}"}},
		},
	})
	raw := userinfoMap(t, srv, "acme", authorizeCode(t, srv, "acme", false))
	if raw["badge"] != "ok" || raw["handle"] != "alice" || raw["id"] != "acme:alice" {
		t.Fatalf("%v", raw)
	}
}

func TestFacebookMeAlias(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
		Profiles: map[string]profiles.Profile{
			"facebook": {
				Template:  "facebook",
				Endpoints: map[string]string{"userinfo": "/me"},
			},
		},
	})
	code := authorizeCode(t, srv, "facebook", false)
	tokenJSON := exchangeToken(t, srv, "facebook", code, http.StatusOK)
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(tokenJSON), &tok); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/facebook/me", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	pic, _ := raw["picture"].(map[string]any)
	data, _ := pic["data"].(map[string]any)
	if data["url"] != oauth.Alice.Avatar {
		t.Fatalf("picture = %v", raw["picture"])
	}
}

func TestAppleProtocolFormPostAndIDToken(t *testing.T) {
	t.Parallel()
	srv := mustServer(t, Options{
		Addr:        listen.Addr{Host: "127.0.0.1", Port: 4190},
		AutoApprove: true,
		Profiles: map[string]profiles.Profile{
			"apple": {
				Template: "apple",
				Protocol: profiles.Protocol{ResponseMode: "form_post", IDToken: true},
			},
		},
	})
	q := url.Values{
		"client_id":     {"dev-client"},
		"redirect_uri":  {"http://127.0.0.1:9999/callback"},
		"response_type": {"code"},
		"state":         {"state-xyz"},
		"auto":          {"alice"},
	}
	req := httptest.NewRequest(http.MethodGet, "/apple/authorize?"+q.Encode(), nil)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK || !strings.Contains(rec.Body.String(), `method="post"`) {
		t.Fatalf("status = %d body = %s", rec.Code, rec.Body.String())
	}
	_, values := extractCallbackForm(t, rec.Body.String())
	body := exchangeToken(t, srv, "apple", values.Get("code"), http.StatusOK)
	if !strings.Contains(body, `"id_token"`) {
		t.Fatalf("apple protocol should issue id_token: %s", body)
	}
}

func userinfoMap(t *testing.T, srv *Server, provider, code string) map[string]any {
	t.Helper()
	tokenJSON := exchangeToken(t, srv, provider, code, http.StatusOK)
	var tok struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal([]byte(tokenJSON), &tok); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodGet, "/"+provider+"/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+tok.AccessToken)
	rec := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("userinfo status = %d body = %s", rec.Code, rec.Body.String())
	}
	var raw map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
		t.Fatal(err)
	}
	return raw
}
