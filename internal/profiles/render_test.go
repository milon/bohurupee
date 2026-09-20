package profiles

import (
	"testing"

	"github.com/milon/bohurupee/internal/oauth"
)

func TestRenderGenericWithoutProfile(t *testing.T) {
	t.Parallel()
	body := Render("acme", oauth.Alice, Profile{})
	if body["id"] != "acme:alice" || body["login"] != "alice" || body["preferred_username"] != "alice" {
		t.Fatalf("default template: %v", body)
	}
	if body["html_url"] != nil || body["email"] != oauth.Alice.Email {
		t.Fatalf("default should not look like github: %v", body)
	}

	bare := Render("acme", oauth.Alice, Profile{Template: "generic"})
	if bare["id"] != "acme:alice" || bare["login"] != nil {
		t.Fatalf("generic opt-out: %v", bare)
	}
}

func TestRenderGitHubTemplate(t *testing.T) {
	t.Parallel()
	body := Render("github", oauth.Alice, Profile{})
	if body["login"] != "alice" || body["avatar_url"] != oauth.Alice.Avatar || body["html_url"] != "https://github.com/alice" {
		t.Fatalf("slug match: %v", body)
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

func TestRenderMoreTemplates(t *testing.T) {
	t.Parallel()

	jump := Render("jumpcloud", oauth.Alice, Profile{Template: "jumpcloud"})
	groups, _ := jump["member_of"].([]any)
	if jump["preferred_username"] != "alice" || jump["jc_org"] != "dev-org" || len(groups) != 1 || groups[0] != "Everyone" {
		t.Fatalf("jumpcloud: %v", jump)
	}
	if jump["id"] != "jumpcloud:alice" || jump["email"] != oauth.Alice.Email {
		t.Fatalf("jumpcloud getters missing: %v", jump)
	}

	gl := Render("gitlab", oauth.Alice, Profile{Template: "gitlab"})
	if gl["username"] != "alice" || gl["web_url"] != "https://gitlab.com/alice" || gl["name"] != oauth.Alice.Name {
		t.Fatalf("gitlab: %v", gl)
	}

	bb := Render("bitbucket", oauth.Alice, Profile{Template: "bitbucket"})
	links, _ := bb["links"].(map[string]any)
	avatar, _ := links["avatar"].(map[string]any)
	if bb["display_name"] != oauth.Alice.Name || avatar["href"] != oauth.Alice.Avatar || bb["email"] != oauth.Alice.Email {
		t.Fatalf("bitbucket: %v", bb)
	}

	slack := Render("slack", oauth.Alice, Profile{Template: "slack"})
	user, _ := slack["user"].(map[string]any)
	profile, _ := user["profile"].(map[string]any)
	if user["name"] != "alice" || profile["image_192"] != oauth.Alice.Avatar || slack["id"] != "slack:alice" {
		t.Fatalf("slack: %v", slack)
	}

	li := Render("linkedin", oauth.Alice, Profile{Template: "linkedin"})
	if li["given_name"] != "Alice" || li["family_name"] != "Admin" || li["picture"] != oauth.Alice.Avatar {
		t.Fatalf("linkedin: %v", li)
	}

	dc := Render("discord", oauth.Alice, Profile{Template: "discord"})
	if dc["username"] != "alice" || dc["global_name"] != oauth.Alice.Name || dc["nickname"] != "alice" {
		t.Fatalf("discord: %v", dc)
	}

	ms := Render("microsoft", oauth.Alice, Profile{Template: "microsoft"})
	if ms["displayName"] != oauth.Alice.Name || ms["mail"] != oauth.Alice.Email || ms["userPrincipalName"] != oauth.Alice.Email {
		t.Fatalf("microsoft: %v", ms)
	}
}

func TestUnknownTemplateRejected(t *testing.T) {
	t.Parallel()
	_, err := NewRegistry(map[string]Profile{"x": {Template: "nope"}})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRenderPersonaOverlayWins(t *testing.T) {
	t.Parallel()
	bob := oauth.Persona{
		ID:            "bob",
		Email:         "bob@example.com",
		EmailVerified: false,
		Name:          "Bob User",
		Nickname:      "bob",
		Avatar:        "",
		Claims:        map[string]any{"role": "user"},
		Response:      map[string]any{"picture": ""},
	}
	body := Render("acme", bob, Profile{})
	if body["email_verified"] != false || body["role"] != "user" {
		t.Fatalf("%v", body)
	}
	if body["avatar"] != "" || body["picture"] != "" {
		t.Fatalf("empty overlay lost: %v", body)
	}
	if body["id"] != "acme:bob" || body["email"] != bob.Email {
		t.Fatalf("getters missing: %v", body)
	}
}
