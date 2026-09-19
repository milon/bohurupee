package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

func VerifyPKCE(method, challenge, verifier string) error {
	if challenge == "" {
		return nil
	}
	if verifier == "" {
		return fmt.Errorf("missing code_verifier")
	}
	switch method {
	case "plain":
		if verifier != challenge {
			return fmt.Errorf("code_verifier mismatch")
		}
		return nil
	case "S256", "":
		sum := sha256.Sum256([]byte(verifier))
		got := base64.RawURLEncoding.EncodeToString(sum[:])
		if got != challenge {
			return fmt.Errorf("code_verifier mismatch")
		}
		return nil
	default:
		return fmt.Errorf("unsupported code_challenge_method")
	}
}
