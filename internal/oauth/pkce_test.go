package oauth

import (
	"crypto/sha256"
	"encoding/base64"
	"testing"
)

func TestVerifyPKCES256(t *testing.T) {
	t.Parallel()
	verifier := "dBjftJeZ4CVP-mB92K27uhbUJU1p1r_wW1gFWFOEjXk"
	sum := sha256.Sum256([]byte(verifier))
	challenge := base64.RawURLEncoding.EncodeToString(sum[:])
	if err := VerifyPKCE("S256", challenge, verifier); err != nil {
		t.Fatal(err)
	}
	if err := VerifyPKCE("S256", challenge, "wrong"); err == nil {
		t.Fatal("expected mismatch")
	}
}

func TestCheckAuthorizePKCE(t *testing.T) {
	t.Parallel()
	if err := CheckAuthorizePKCE(PKCERequired, ""); err == nil {
		t.Fatal("required should reject missing challenge")
	}
	if err := CheckAuthorizePKCE(PKCEForbidden, "abc"); err == nil {
		t.Fatal("forbidden should reject challenge")
	}
	if err := CheckAuthorizePKCE(PKCEOptional, ""); err != nil {
		t.Fatal(err)
	}
}
