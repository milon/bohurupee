package profiles

import (
	"strings"

	"github.com/milon/bohurupee/internal/oauth"
)

type TemplateFunc func(provider string, persona oauth.Persona) map[string]any

func Template(name string) TemplateFunc {
	return templates[strings.ToLower(strings.TrimSpace(name))]
}

var templates = map[string]TemplateFunc{
	"github":   githubTemplate,
	"google":   googleTemplate,
	"facebook": facebookTemplate,
	"twitch":   twitchTemplate,
	"apple":    appleTemplate,
}

func githubTemplate(_ string, p oauth.Persona) map[string]any {
	login := p.Nickname
	return map[string]any{
		"login":      login,
		"avatar_url": p.Avatar,
		"html_url":   "https://github.com/" + login,
		"type":       "User",
	}
}

func googleTemplate(_ string, p oauth.Persona) map[string]any {
	given, family := splitName(p.Name)
	return map[string]any{
		"picture":        p.Avatar,
		"given_name":     given,
		"family_name":    family,
		"verified_email": p.EmailVerified,
		"email_verified": p.EmailVerified,
	}
}

func facebookTemplate(_ string, p oauth.Persona) map[string]any {
	return map[string]any{
		"picture": map[string]any{
			"data": map[string]any{
				"url":           p.Avatar,
				"is_silhouette": false,
			},
		},
	}
}

func twitchTemplate(_ string, p oauth.Persona) map[string]any {
	return map[string]any{
		"login":             p.Nickname,
		"display_name":      p.Name,
		"profile_image_url": p.Avatar,
	}
}

func appleTemplate(_ string, p oauth.Persona) map[string]any {
	return map[string]any{
		"email_verified": p.EmailVerified,
	}
}

func splitName(name string) (given, family string) {
	parts := strings.Fields(name)
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], strings.Join(parts[1:], " ")
	}
}
