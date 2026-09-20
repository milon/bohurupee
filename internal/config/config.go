package config

import (
	_ "embed"
	"fmt"
	"os"
	"strings"

	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/oidc"
	"github.com/milon/bohurupee/internal/profiles"
	"gopkg.in/yaml.v3"
)

//go:embed example.yaml
var defaultYAML []byte

const DefaultPath = "bohurupee.yaml"

type File struct {
	Port          int                    `yaml:"port"`
	Bind          string                 `yaml:"bind"`
	PKCE          string                 `yaml:"pkce"`
	IDToken       string                 `yaml:"idToken"`
	OpenClient    *bool                  `yaml:"openClient"`
	RefreshTokens *bool                  `yaml:"refreshTokens"`
	Personas      []filePersona          `yaml:"personas"`
	Clients       []fileClient           `yaml:"clients"`
	Profiles      map[string]fileProfile `yaml:"providerProfiles"`
}

type fileClient struct {
	ID           string   `yaml:"id"`
	RedirectURIs []string `yaml:"redirect_uris"`
}

type fileProfile struct {
	ResponseTemplate string            `yaml:"responseTemplate"`
	Response         map[string]any    `yaml:"response"`
	Endpoints        map[string]string `yaml:"endpoints"`
	Protocol         fileProtocol      `yaml:"protocol"`
}

type fileProtocol struct {
	ResponseMode string `yaml:"response_mode"`
	IDToken      *bool  `yaml:"id_token"`
}

type filePersona struct {
	ID            string         `yaml:"id"`
	Email         string         `yaml:"email"`
	EmailVerified *bool          `yaml:"email_verified"`
	Name          string         `yaml:"name"`
	Nickname      string         `yaml:"nickname"`
	Avatar        *string        `yaml:"avatar"`
	Claims        map[string]any `yaml:"claims"`
	Response      map[string]any `yaml:"response"`
}

type Config struct {
	Port          int
	Bind          string
	BindFromFile  bool
	PKCE          oauth.PKCEMode
	IDToken       oidc.IDTokenMode
	OpenClient    bool
	RefreshTokens bool
	Personas      []oauth.Persona
	Clients       map[string]Client
	Profiles      map[string]profiles.Profile
}

// Client is an optional registered OAuth client with an allowlisted redirect set.
type Client struct {
	ID           string
	RedirectURIs []string
}

// Defaults is the built-in config (same content as bohurupee init / example.yaml).
// A user YAML file is a sparse overlay on top of this.
func Defaults() Config {
	cfg := Config{
		Port:       4190,
		Bind:       "127.0.0.1",
		PKCE:       oauth.PKCEOptional,
		IDToken:    oidc.IDTokenOpenID,
		OpenClient: true,
		Personas:   []oauth.Persona{oauth.Alice},
	}
	if err := overlayYAML(&cfg, defaultYAML, false); err != nil {
		panic("config: embedded example.yaml: " + err.Error())
	}
	// The embed is not a user-provided file; listen overrides stay unset.
	cfg.BindFromFile = false
	return cfg
}

// LoadPath reads YAML from path and overlays it on Defaults. Omitted keys keep
// their default values, except personas, which must be set in every user file.
// Empty path loads Defaults, then overlays DefaultPath in the working
// directory when that file exists.
func LoadPath(path string) (Config, error) {
	cfg := Defaults()
	if path == "" {
		if _, err := os.Stat(DefaultPath); err != nil {
			if os.IsNotExist(err) {
				return cfg, nil
			}
			return Config{}, err
		}
		path = DefaultPath
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := overlayYAML(&cfg, raw, true); err != nil {
		return Config{}, fmt.Errorf("config %q: %w", path, err)
	}
	return cfg, nil
}

func overlayYAML(cfg *Config, raw []byte, requirePersonas bool) error {
	var f File
	if err := yaml.Unmarshal(raw, &f); err != nil {
		return err
	}
	if f.Port != 0 {
		cfg.Port = f.Port
	}
	if strings.TrimSpace(f.Bind) != "" {
		cfg.Bind = strings.TrimSpace(f.Bind)
		cfg.BindFromFile = true
	}
	if strings.TrimSpace(f.PKCE) != "" {
		mode, err := oauth.ParsePKCEMode(f.PKCE)
		if err != nil {
			return err
		}
		cfg.PKCE = mode
	}
	if strings.TrimSpace(f.IDToken) != "" {
		mode, err := oidc.ParseIDTokenMode(f.IDToken)
		if err != nil {
			return err
		}
		cfg.IDToken = mode
	}
	if f.OpenClient != nil {
		cfg.OpenClient = *f.OpenClient
	}
	if f.RefreshTokens != nil {
		cfg.RefreshTokens = *f.RefreshTokens
	}
	if f.Clients != nil {
		out := make(map[string]Client, len(f.Clients))
		for i, fc := range f.Clients {
			id := strings.TrimSpace(fc.ID)
			if id == "" {
				return fmt.Errorf("clients[%d]: id is required", i)
			}
			if _, dup := out[id]; dup {
				return fmt.Errorf("clients: duplicate id %q", id)
			}
			uris := make([]string, 0, len(fc.RedirectURIs))
			for _, raw := range fc.RedirectURIs {
				u := strings.TrimSpace(raw)
				if u == "" {
					continue
				}
				uris = append(uris, u)
			}
			out[id] = Client{ID: id, RedirectURIs: uris}
		}
		cfg.Clients = out
	}
	if f.Personas == nil {
		if requirePersonas {
			return fmt.Errorf("personas is required")
		}
	} else {
		if len(f.Personas) == 0 {
			return fmt.Errorf("personas must not be empty")
		}
		ps := make([]oauth.Persona, 0, len(f.Personas))
		for _, fp := range f.Personas {
			p, err := fp.toPersona()
			if err != nil {
				return err
			}
			ps = append(ps, p)
		}
		cfg.Personas = ps
	}
	if f.Profiles != nil {
		if cfg.Profiles == nil {
			cfg.Profiles = make(map[string]profiles.Profile)
		}
		for name, fp := range f.Profiles {
			key := strings.ToLower(strings.TrimSpace(name))
			merged := mergeFileProfile(cfg.Profiles[key], fp)
			switch merged.Protocol.ResponseMode {
			case "", "query", "form_post":
			default:
				return fmt.Errorf("providerProfiles.%s: protocol.response_mode must be query or form_post", name)
			}
			cfg.Profiles[key] = merged
		}
		if _, err := profiles.NewRegistry(cfg.Profiles); err != nil {
			return err
		}
	}
	return nil
}

func mergeFileProfile(base profiles.Profile, fp fileProfile) profiles.Profile {
	out := base
	if t := strings.TrimSpace(fp.ResponseTemplate); t != "" {
		out.Template = t
	}
	if fp.Response != nil {
		out.Response = overlayAnyMap(out.Response, fp.Response)
	}
	if fp.Endpoints != nil {
		if out.Endpoints == nil {
			out.Endpoints = make(map[string]string, len(fp.Endpoints))
		} else {
			cp := make(map[string]string, len(out.Endpoints)+len(fp.Endpoints))
			for k, v := range out.Endpoints {
				cp[k] = v
			}
			out.Endpoints = cp
		}
		for k, v := range fp.Endpoints {
			out.Endpoints[k] = v
		}
	}
	if mode := strings.TrimSpace(fp.Protocol.ResponseMode); mode != "" {
		out.Protocol.ResponseMode = mode
	}
	if fp.Protocol.IDToken != nil {
		out.Protocol.IDToken = *fp.Protocol.IDToken
	}
	return out
}

func overlayAnyMap(base, over map[string]any) map[string]any {
	if len(base) == 0 {
		if over == nil {
			return nil
		}
		out := make(map[string]any, len(over))
		for k, v := range over {
			out[k] = v
		}
		return out
	}
	out := make(map[string]any, len(base)+len(over))
	for k, v := range base {
		out[k] = v
	}
	for k, v := range over {
		if bm, ok := out[k].(map[string]any); ok {
			if om, ok := v.(map[string]any); ok {
				out[k] = overlayAnyMap(bm, om)
				continue
			}
		}
		out[k] = v
	}
	return out
}

func (fp filePersona) toPersona() (oauth.Persona, error) {
	id := strings.TrimSpace(fp.ID)
	if !oauth.ValidProvider(id) {
		return oauth.Persona{}, fmt.Errorf("invalid persona id %q", fp.ID)
	}
	verified := true
	if fp.EmailVerified != nil {
		verified = *fp.EmailVerified
	}
	nick := strings.TrimSpace(fp.Nickname)
	if nick == "" {
		nick = id
	}
	avatar := "https://api.dicebear.com/9.x/identicon/svg?seed=" + id
	if fp.Avatar != nil {
		avatar = strings.TrimSpace(*fp.Avatar)
	}
	return oauth.Persona{
		ID:            id,
		Email:         strings.TrimSpace(fp.Email),
		EmailVerified: verified,
		Name:          strings.TrimSpace(fp.Name),
		Nickname:      nick,
		Avatar:        avatar,
		Claims:        fp.Claims,
		Response:      fp.Response,
	}, nil
}
