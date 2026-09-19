package oauth

import "strings"

// Alice is the hardcoded persona. YAML personas arrive later.
var Alice = Persona{
	ID:            "alice",
	Email:         "alice@example.com",
	EmailVerified: true,
	Name:          "Alice Admin",
	Nickname:      "alice",
	Avatar:        "https://api.dicebear.com/9.x/identicon/svg?seed=alice",
}

type Persona struct {
	ID            string
	Email         string
	EmailVerified bool
	Name          string
	Nickname      string
	Avatar        string
}

type Userinfo struct {
	ID            string `json:"id"`
	Sub           string `json:"sub"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"email_verified"`
	Name          string `json:"name"`
	Nickname      string `json:"nickname"`
	Avatar        string `json:"avatar"`
}

func (p Persona) Userinfo(provider string) Userinfo {
	id := StableID(provider, p.ID)
	return Userinfo{
		ID:            id,
		Sub:           id,
		Email:         p.Email,
		EmailVerified: p.EmailVerified,
		Name:          p.Name,
		Nickname:      p.Nickname,
		Avatar:        p.Avatar,
	}
}

func StableID(provider, personaID string) string {
	return provider + ":" + personaID
}

func PersonaByID(id string) (Persona, bool) {
	if strings.EqualFold(id, Alice.ID) {
		return Alice, true
	}
	return Persona{}, false
}
