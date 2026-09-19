package version

import "strings"

// Version is the release version, injected by GoReleaser:
//
//	-X github.com/milon/bohurupee/internal/version.Version={{.Version}}
//
// Local builds stay "dev".
var Version = "dev"

func String() string {
	return format(Version)
}

func format(v string) string {
	v = strings.TrimSpace(v)
	switch {
	case v == "", v == "dev":
		return "dev"
	case strings.HasPrefix(v, "v"):
		return v
	default:
		return "v" + v
	}
}
