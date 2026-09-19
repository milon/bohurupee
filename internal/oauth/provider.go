package oauth

import "unicode"

// ValidProvider is a URL slug: starts with a letter or digit, then letters,
// digits, underscore, or hyphen. Keeps /{provider}/authorize distinct from
// later discovery paths without a router.
func ValidProvider(slug string) bool {
	if slug == "" {
		return false
	}
	for i, r := range slug {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
		case i > 0 && (r == '-' || r == '_'):
		default:
			return false
		}
	}
	return true
}
