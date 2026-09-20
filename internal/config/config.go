package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/milon/bohurupee/internal/oauth"
	"github.com/milon/bohurupee/internal/oidc"
	"github.com/milon/bohurupee/internal/profiles"
	"gopkg.in/yaml.v3"
)

const DefaultPath = "bohurupee.yaml"

type File struct {
	Port          int                    `yaml:"port"`
	Bind          string                 `yaml:"bind"`
	PKCE          string                 `yaml:"pkce"`
	IDToken       string                 `yaml:"idToken"`
	OpenClient    *bool                  `yaml:"openClient"`
	RefreshTokens *bool                  `yaml:"refreshTokens"`
	Personas      []filePersona          `yaml:"personas"`
	Profiles      map[string]fileProfile `yaml:"providerProfiles"`
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
	Profiles      map[string]profiles.Profile
}

func Defaults() Config {
	return Config{
		Port:       4190,
		Bind:       "127.0.0.1",
		PKCE:       oauth.PKCEOptional,
		IDToken:    oidc.IDTokenOpenID,
		OpenClient: true,
		Personas:   []oauth.Persona{oauth.Alice},
	}
}

// LoadPath reads YAML from path. Empty path loads Defaults, then overlays
// DefaultPath in the working directory when that file exists.
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
	if err := overlayYAML(&cfg, raw); err != nil {
		return Config{}, fmt.Errorf("config %q: %w", path, err)
	}
	return cfg, nil
}

func overlayYAML(cfg *Config, raw []byte) error {
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
	if f.Personas != nil {
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
		out := make(map[string]profiles.Profile, len(f.Profiles))
		for name, fp := range f.Profiles {
			p := profiles.Profile{
				Template:  strings.TrimSpace(fp.ResponseTemplate),
				Response:  fp.Response,
				Endpoints: fp.Endpoints,
				Protocol: profiles.Protocol{
					ResponseMode: strings.TrimSpace(fp.Protocol.ResponseMode),
				},
			}
			if fp.Protocol.IDToken != nil {
				p.Protocol.IDToken = *fp.Protocol.IDToken
			}
			switch p.Protocol.ResponseMode {
			case "", "query", "form_post":
			default:
				return fmt.Errorf("providerProfiles.%s: protocol.response_mode must be query or form_post", name)
			}
			out[name] = p
		}
		if _, err := profiles.NewRegistry(out); err != nil {
			return err
		}
		cfg.Profiles = out
	}
	return nil
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
