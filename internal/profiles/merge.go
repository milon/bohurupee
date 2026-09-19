package profiles

import (
	"fmt"
	"strings"

	"github.com/milon/bohurupee/internal/oauth"
)

func mergeMaps(base, over map[string]any) map[string]any {
	out := cloneMap(base)
	for k, v := range over {
		if bm, ok := out[k].(map[string]any); ok {
			if om, ok := v.(map[string]any); ok {
				out[k] = mergeMaps(bm, om)
				continue
			}
		}
		out[k] = cloneValue(v)
	}
	return out
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = cloneValue(v)
	}
	return out
}

func cloneValue(v any) any {
	switch t := v.(type) {
	case map[string]any:
		return cloneMap(t)
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = cloneValue(item)
		}
		return out
	default:
		return v
	}
}

func interpolateMap(v map[string]any, provider string, persona oauth.Persona) map[string]any {
	out, _ := interpolate(v, provider, persona).(map[string]any)
	if out == nil {
		return map[string]any{}
	}
	return out
}

func interpolate(v any, provider string, persona oauth.Persona) any {
	switch t := v.(type) {
	case string:
		return expand(t, provider, persona)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, item := range t {
			out[k] = interpolate(item, provider, persona)
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, item := range t {
			out[i] = interpolate(item, provider, persona)
		}
		return out
	default:
		return v
	}
}

func expand(s, provider string, persona oauth.Persona) string {
	u := persona.Userinfo(provider)
	repl := map[string]string{
		"id":       u.ID,
		"sub":      u.Sub,
		"email":    u.Email,
		"name":     u.Name,
		"nickname": u.Nickname,
		"avatar":   u.Avatar,
		"provider": provider,
	}
	out := s
	for k, v := range repl {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
		out = strings.ReplaceAll(out, fmt.Sprintf("{{ %s }}", k), v)
	}
	return out
}
