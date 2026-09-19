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
	"github":    githubTemplate,
	"google":    googleTemplate,
	"facebook":  facebookTemplate,
	"twitch":    twitchTemplate,
	"apple":     appleTemplate,
	"jumpcloud": jumpcloudTemplate,
	"gitlab":    gitlabTemplate,
	"bitbucket": bitbucketTemplate,
	"slack":     slackTemplate,
	"linkedin":  linkedinTemplate,
	"discord":   discordTemplate,
	"microsoft": microsoftTemplate,
	"default":   defaultTemplate,
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

// defaultTemplate covers Socialite drivers that have no built-in shape.
// The keys are the aliases those drivers read (login, username, picture, …).
func defaultTemplate(_ string, p oauth.Persona) map[string]any {
	given, family := splitName(p.Name)
	return map[string]any{
		"login":              p.Nickname,
		"username":           p.Nickname,
		"preferred_username": p.Nickname,
		"display_name":       p.Name,
		"given_name":         given,
		"family_name":        family,
		"picture":            p.Avatar,
		"avatar_url":         p.Avatar,
		"profile_image_url":  p.Avatar,
	}
}

func jumpcloudTemplate(_ string, p oauth.Persona) map[string]any {
	given, family := splitName(p.Name)
	return map[string]any{
		"preferred_username": p.Nickname,
		"given_name":         given,
		"family_name":        family,
		"middle_name":        "",
		"jc_org":             "dev-org",
		"member_of":          []any{"Everyone"},
	}
}

func gitlabTemplate(_ string, p oauth.Persona) map[string]any {
	return map[string]any{
		"username":   p.Nickname,
		"avatar_url": p.Avatar,
		"web_url":    "https://gitlab.com/" + p.Nickname,
		"state":      "active",
	}
}

func bitbucketTemplate(_ string, p oauth.Persona) map[string]any {
	return map[string]any{
		"username":     p.Nickname,
		"display_name": p.Name,
		"uuid":         "{" + p.ID + "}",
		"type":         "user",
		"links": map[string]any{
			"avatar": map[string]any{"href": p.Avatar},
		},
	}
}

func slackTemplate(_ string, p oauth.Persona) map[string]any {
	return map[string]any{
		"ok": true,
		"user": map[string]any{
			"name":      p.Nickname,
			"real_name": p.Name,
			"profile": map[string]any{
				"email":        p.Email,
				"display_name": p.Nickname,
				"image_192":    p.Avatar,
			},
		},
	}
}

func linkedinTemplate(_ string, p oauth.Persona) map[string]any {
	given, family := splitName(p.Name)
	return map[string]any{
		"given_name":  given,
		"family_name": family,
		"picture":     p.Avatar,
	}
}

func discordTemplate(_ string, p oauth.Persona) map[string]any {
	return map[string]any{
		"username":      p.Nickname,
		"global_name":   p.Name,
		"discriminator": "0",
	}
}

func microsoftTemplate(_ string, p oauth.Persona) map[string]any {
	given, family := splitName(p.Name)
	return map[string]any{
		"displayName":       p.Name,
		"givenName":         given,
		"surname":           family,
		"mail":              p.Email,
		"userPrincipalName": p.Email,
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
