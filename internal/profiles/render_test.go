package profiles

import (
	"testing"

	"github.com/milon/bohurupee/internal/oauth"
)

func TestRenderGenericWithoutProfile(t *testing.T) {
	t.Parallel()
	body := Render("acme", oauth.Alice, Profile{})
	if body["id"] != "acme:alice" || body["login"] != nil {
		t.Fatalf("%v", body)
	}
}

func TestRenderGitHubTemplate(t *testing.T) {
	t.Parallel()
	body := Render("github", oauth.Alice, Profile{Template: "github"})
	if body["login"] != "alice" || body["avatar_url"] != oauth.Alice.Avatar {
		t.Fatalf("%v", body)
	}
	if body["id"] != "github:alice" || body["email"] != oauth.Alice.Email {
		t.Fatalf("getters missing: %v", body)
	}
}

func TestRenderCustomResponse(t *testing.T) {
	t.Parallel()
	body := Render("acme", oauth.Alice, Profile{
		Response: map[string]any{
			"title":    "Staff",
			"username": "{{nickname}}",
		},
	})
	if body["title"] != "Staff" || body["username"] != "alice" {
		t.Fatalf("%v", body)
	}
	if body["email"] != oauth.Alice.Email {
		t.Fatalf("persona overlay lost: %v", body)
	}
}

func TestUnknownTemplateRejected(t *testing.T) {
	t.Parallel()
	_, err := NewRegistry(map[string]Profile{"x": {Template: "nope"}})
	if err == nil {
		t.Fatal("expected error")
	}
}
