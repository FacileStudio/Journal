package apikeys

import (
	"net/url"
	"slices"
	"strings"

	"github.com/FacileStudio/tronc/errors"
)

// NormalizeOrigins turns what an admin typed into the exact strings a browser
// puts in the Origin header: scheme and host lowercased, the default port
// dropped, no path and no trailing slash.
//
// It is strict on purpose. An allowlist entry that never matches is an outage
// that looks like a bug in the SDK, and one that matches more than it should
// is the reason the allowlist exists.
func NormalizeOrigins(origins []string) ([]string, error) {
	if len(origins) == 0 {
		return nil, errors.Invalid("a public key needs at least one allowed origin")
	}
	if len(origins) > maxAllowedOrigin {
		return nil, errors.Invalid("a public key accepts at most 8 allowed origins")
	}

	normalized := make([]string, 0, len(origins))
	for _, raw := range origins {
		origin, err := NormalizeOrigin(raw)
		if err != nil {
			return nil, err
		}
		if !slices.Contains(normalized, origin) {
			normalized = append(normalized, origin)
		}
	}
	return normalized, nil
}

// NormalizeOrigin trims and validates an allowed-origin string, rejecting
// empty values.
func NormalizeOrigin(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", errors.Invalid("an allowed origin cannot be empty")
	}
	parsed, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.Invalid("allowed origin " + trimmed + " is not a URL")
	}

	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return "", errors.Invalid("allowed origin " + trimmed + " must start with http:// or https://")
	}
	if parsed.Hostname() == "" {
		return "", errors.Invalid("allowed origin " + trimmed + " has no host")
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return "", errors.Invalid("allowed origin " + trimmed + " must carry no path")
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" || parsed.User != nil {
		return "", errors.Invalid("allowed origin " + trimmed + " must be scheme://host[:port] only")
	}
	if strings.Contains(parsed.Hostname(), "*") {
		return "", errors.Invalid("wildcards are not accepted in an allowed origin; list each host")
	}

	origin := scheme + "://" + strings.ToLower(parsed.Hostname())
	port := parsed.Port()
	if port != "" && !isDefaultPort(scheme, port) {
		origin += ":" + port
	}
	return origin, nil
}

func isDefaultPort(scheme, port string) bool {
	return (scheme == "http" && port == "80") || (scheme == "https" && port == "443")
}
