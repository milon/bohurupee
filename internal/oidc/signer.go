package oidc

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/lestrrat-go/jwx/v3/jwt"
	"github.com/milon/bohurupee/internal/oauth"
)

type IDTokenMode string

const (
	IDTokenOpenID IDTokenMode = "openid"
	IDTokenAlways IDTokenMode = "always"
)

func ParseIDTokenMode(s string) (IDTokenMode, error) {
	switch IDTokenMode(strings.ToLower(strings.TrimSpace(s))) {
	case "", IDTokenOpenID:
		return IDTokenOpenID, nil
	case IDTokenAlways:
		return IDTokenAlways, nil
	default:
		return "", fmt.Errorf("idToken must be openid or always (got %q)", s)
	}
}

func WantIDToken(mode IDTokenMode, scope string) bool {
	if mode == IDTokenAlways {
		return true
	}
	return oauth.HasScope(scope, "openid")
}

type Signer struct {
	raw *rsa.PrivateKey
	key jwk.Key
	pub jwk.Set
}

func Generate() (*Signer, error) {
	raw, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate oidc key: %w", err)
	}
	return fromRSA(raw)
}

func LoadOrCreate(path string) (*Signer, error) {
	if path == "" {
		return Generate()
	}
	raw, err := os.ReadFile(path)
	if err == nil {
		return ParsePEM(raw)
	}
	if !os.IsNotExist(err) {
		return nil, fmt.Errorf("read oidc key: %w", err)
	}
	s, err := Generate()
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return s, nil
	}
	pemBytes, err := s.MarshalPKCS1()
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(path, pemBytes, 0o600); err != nil {
		return s, nil
	}
	return s, nil
}

func DefaultKeyPath() string {
	dir, err := os.UserConfigDir()
	if err != nil {
		return ""
	}
	return filepath.Join(dir, "bohurupee", "oidc.key")
}

func ParsePEM(pemBytes []byte) (*Signer, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("oidc key: no PEM block")
	}
	var key *rsa.PrivateKey
	var err error
	switch block.Type {
	case "RSA PRIVATE KEY":
		key, err = x509.ParsePKCS1PrivateKey(block.Bytes)
	case "PRIVATE KEY":
		parsed, perr := x509.ParsePKCS8PrivateKey(block.Bytes)
		if perr != nil {
			return nil, fmt.Errorf("parse oidc key: %w", perr)
		}
		var ok bool
		key, ok = parsed.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("oidc key: not RSA")
		}
	default:
		return nil, fmt.Errorf("oidc key: unsupported PEM type %q", block.Type)
	}
	if err != nil {
		return nil, fmt.Errorf("parse oidc key: %w", err)
	}
	return fromRSA(key)
}

func fromRSA(raw *rsa.PrivateKey) (*Signer, error) {
	kid := keyID(&raw.PublicKey)
	priv, err := jwk.Import(raw)
	if err != nil {
		return nil, err
	}
	if err := priv.Set(jwk.KeyIDKey, kid); err != nil {
		return nil, err
	}
	if err := priv.Set(jwk.AlgorithmKey, jwa.RS256()); err != nil {
		return nil, err
	}
	if err := priv.Set(jwk.KeyUsageKey, "sig"); err != nil {
		return nil, err
	}
	pub, err := priv.PublicKey()
	if err != nil {
		return nil, err
	}
	set := jwk.NewSet()
	if err := set.AddKey(pub); err != nil {
		return nil, err
	}
	return &Signer{raw: raw, key: priv, pub: set}, nil
}

func (s *Signer) MarshalPKCS1() ([]byte, error) {
	der := x509.MarshalPKCS1PrivateKey(s.raw)
	return pem.EncodeToMemory(&pem.Block{Type: "RSA PRIVATE KEY", Bytes: der}), nil
}

func (s *Signer) JWKS() ([]byte, error) {
	return json.Marshal(s.pub)
}

type IDTokenInput struct {
	Issuer   string
	Audience string
	Nonce    string
	Now      time.Time
	TTL      time.Duration
	Provider string
	Persona  oauth.Persona
}

func (s *Signer) IDToken(in IDTokenInput) (string, error) {
	if in.TTL <= 0 {
		in.TTL = oauth.DefaultTokenTTL
	}
	tok := jwt.New()
	sub := oauth.StableID(in.Provider, in.Persona.ID)
	for _, kv := range []struct {
		k string
		v any
	}{
		{jwt.IssuerKey, in.Issuer},
		{jwt.SubjectKey, sub},
		{jwt.AudienceKey, []string{in.Audience}},
		{jwt.IssuedAtKey, in.Now},
		{jwt.ExpirationKey, in.Now.Add(in.TTL)},
		{"email", in.Persona.Email},
		{"email_verified", in.Persona.EmailVerified},
		{"name", in.Persona.Name},
		{"nickname", in.Persona.Nickname},
		{"picture", in.Persona.Avatar},
	} {
		if err := tok.Set(kv.k, kv.v); err != nil {
			return "", err
		}
	}
	if in.Nonce != "" {
		if err := tok.Set("nonce", in.Nonce); err != nil {
			return "", err
		}
	}
	signed, err := jwt.Sign(tok, jwt.WithKey(jwa.RS256(), s.key))
	if err != nil {
		return "", err
	}
	return string(signed), nil
}

func keyID(pub *rsa.PublicKey) string {
	der, err := x509.MarshalPKIXPublicKey(pub)
	if err != nil {
		return "dev"
	}
	sum := sha256.Sum256(der)
	return hex.EncodeToString(sum[:8])
}
