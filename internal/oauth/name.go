package oauth

import "strings"

// SplitName splits a display name into given and family parts.
// "Alice Admin" → ("Alice", "Admin"); a single token has an empty family.
func SplitName(name string) (given, family string) {
	parts := strings.Fields(name)
	switch len(parts) {
	case 0:
		return "", ""
	case 1:
		return parts[0], ""
	default:
		return parts[0], strings.Join(parts[1:], " ")
	}
}
