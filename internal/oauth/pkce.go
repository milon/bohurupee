package oauth

import (
	"fmt"
	"strings"
)

type PKCEMode string

const (
	PKCEOptional  PKCEMode = "optional"
	PKCERequired  PKCEMode = "required"
	PKCEForbidden PKCEMode = "forbidden"
)

func ParsePKCEMode(s string) (PKCEMode, error) {
	switch PKCEMode(strings.ToLower(strings.TrimSpace(s))) {
	case PKCEOptional, "":
		return PKCEOptional, nil
	case PKCERequired:
		return PKCERequired, nil
	case PKCEForbidden:
		return PKCEForbidden, nil
	default:
		return "", fmt.Errorf("pkce must be optional, required, or forbidden (got %q)", s)
	}
}

func NormalizeChallengeMethod(method, challenge string) (string, error) {
	method = strings.TrimSpace(method)
	if challenge == "" {
		if method != "" {
			return "", fmt.Errorf("code_challenge_method without code_challenge")
		}
		return "", nil
	}
	if method == "" {
		return "S256", nil
	}
	switch method {
	case "S256", "plain":
		return method, nil
	default:
		return "", fmt.Errorf("code_challenge_method must be S256 or plain")
	}
}

func CheckAuthorizePKCE(mode PKCEMode, challenge string) error {
	switch mode {
	case PKCERequired:
		if challenge == "" {
			return fmt.Errorf("code_challenge is required")
		}
	case PKCEForbidden:
		if challenge != "" {
			return fmt.Errorf("PKCE is forbidden")
		}
	}
	return nil
}
