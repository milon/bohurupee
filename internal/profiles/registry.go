package profiles

import (
	"fmt"
	"strings"

	"github.com/milon/bohurupee/internal/oauth"
)

type Protocol struct {
	ResponseMode string
	IDToken      bool
}

type Profile struct {
	Template  string
	Response  map[string]any
	Endpoints map[string]string
	Protocol  Protocol
}

type Registry struct {
	byName map[string]Profile
}

func NewRegistry(in map[string]Profile) (*Registry, error) {
	r := &Registry{byName: make(map[string]Profile, len(in))}
	for name, p := range in {
		key := strings.ToLower(strings.TrimSpace(name))
		if !oauth.ValidProvider(key) {
			return nil, fmt.Errorf("invalid providerProfiles key %q", name)
		}
		if p.Template != "" && Template(p.Template) == nil {
			return nil, fmt.Errorf("providerProfiles.%s: unknown responseTemplate %q", name, p.Template)
		}
		r.byName[key] = p
	}
	return r, nil
}

func (r *Registry) Get(provider string) (Profile, bool) {
	if r == nil {
		return Profile{}, false
	}
	p, ok := r.byName[strings.ToLower(provider)]
	return p, ok
}

func (r *Registry) UserinfoAliases() []string {
	if r == nil {
		return nil
	}
	seen := map[string]bool{}
	var out []string
	for _, p := range r.byName {
		raw := strings.TrimSpace(p.Endpoints["userinfo"])
		if raw == "" {
			continue
		}
		alias := strings.Trim(raw, "/")
		if alias == "" || alias == "userinfo" {
			continue
		}
		if strings.Contains(alias, "/") || !oauth.ValidProvider(alias) {
			continue
		}
		if seen[alias] {
			continue
		}
		seen[alias] = true
		out = append(out, alias)
	}
	return out
}

// Render merges generic → template → custom response → persona overlay
// so Socialite-style getters still see id/email/name/nickname/avatar.
func Render(provider string, persona oauth.Persona, p Profile) map[string]any {
	out := cloneMap(genericMap(persona, provider))
	if fn := Template(p.Template); fn != nil {
		out = mergeMaps(out, fn(provider, persona))
	}
	if len(p.Response) > 0 {
		out = mergeMaps(out, interpolateMap(p.Response, provider, persona))
	}
	return mergeMaps(out, genericMap(persona, provider))
}

func genericMap(persona oauth.Persona, provider string) map[string]any {
	u := persona.Userinfo(provider)
	return map[string]any{
		"id":             u.ID,
		"sub":            u.Sub,
		"email":          u.Email,
		"email_verified": u.EmailVerified,
		"name":           u.Name,
		"nickname":       u.Nickname,
		"avatar":         u.Avatar,
	}
}
