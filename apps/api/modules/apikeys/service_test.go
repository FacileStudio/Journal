package apikeys

import (
	"strings"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
	"github.com/FacileStudio/Journal/apps/api/schemas"
)

func TestValidAppName(t *testing.T) {
	cases := []struct {
		app  string
		want bool
	}{
		{"nuage", true},
		{"a", true},
		{"0abc", true},
		{"my-app-2", true},
		{strings.Repeat("a", 64), true},
		{"", false},
		{"-abc", false},
		{"ABC", false},
		{"my_app", false},
		{"app name", false},
		{strings.Repeat("a", 65), false},
	}
	for _, tc := range cases {
		t.Run(tc.app, func(t *testing.T) {
			if got := validAppName(tc.app); got != tc.want {
				t.Fatalf("validAppName(%q) = %v, want %v", tc.app, got, tc.want)
			}
		})
	}
}

func TestGenerateTokenRoundTrip(t *testing.T) {
	token, prefix, hash, err := generateToken("nuage", schemas.KeyKindSecret)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}

	head := "journal_nuage_"
	if !strings.HasPrefix(token, head) {
		t.Fatalf("token %q lacks prefix %q", token, head)
	}
	random := strings.TrimPrefix(token, head)
	if len(random) != 43 {
		t.Fatalf("random part is %d chars, want 43", len(random))
	}
	if prefix != head+random[:6] {
		t.Fatalf("prefix %q does not match token head", prefix)
	}
	if hash != authcrypto.HashToken(token) {
		t.Fatal("stored hash does not verify against the full token")
	}

	other, _, otherHash, err := generateToken("nuage", schemas.KeyKindSecret)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if token == other || hash == otherHash {
		t.Fatal("two generated tokens collide")
	}
}

// A public token has to be recognisable on sight: it ends up in a bundle and
// in a git diff, and the reviewer who spots journal_pub_ is the one who does
// not open an incident over a leaked credential.
func TestPublicTokenIsMarked(t *testing.T) {
	token, prefix, _, err := generateToken("shop", schemas.KeyKindPublic)
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if !strings.HasPrefix(token, "journal_pub_shop_") {
		t.Fatalf("token %q lacks the public marker", token)
	}
	if !strings.HasPrefix(prefix, "journal_pub_shop_") {
		t.Fatalf("prefix %q lacks the public marker", prefix)
	}
}

func TestNormalizeOrigin(t *testing.T) {
	cases := []struct {
		raw  string
		want string
	}{
		{"https://shop.example", "https://shop.example"},
		{"https://SHOP.example/", "https://shop.example"},
		{"HTTPS://Shop.Example:443", "https://shop.example"},
		{"http://localhost:5173", "http://localhost:5173"},
		{"http://shop.example:80", "http://shop.example"},
		{"https://shop.example:8443", "https://shop.example:8443"},
	}
	for _, tc := range cases {
		t.Run(tc.raw, func(t *testing.T) {
			got, err := NormalizeOrigin(tc.raw)
			if err != nil {
				t.Fatalf("NormalizeOrigin(%q): %v", tc.raw, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeOrigin(%q) = %q, want %q", tc.raw, got, tc.want)
			}
		})
	}

	rejected := []string{
		"",
		"shop.example",
		"ftp://shop.example",
		"https://",
		"https://*.example",
		"https://shop.example/path",
		"https://shop.example?a=b",
		"https://user:pass@shop.example",
	}
	for _, raw := range rejected {
		t.Run("reject "+raw, func(t *testing.T) {
			if _, err := NormalizeOrigin(raw); err == nil {
				t.Fatalf("NormalizeOrigin(%q) was accepted", raw)
			}
		})
	}
}

// An empty allowlist must reject rather than wave everything through: the
// reading that "unset means any" is what turns a public key into an open relay.
func TestNormalizeOriginsRefusesEmpty(t *testing.T) {
	if _, err := NormalizeOrigins(nil); err == nil {
		t.Fatal("an empty allowlist was accepted")
	}
	if _, err := NormalizeOrigins(make([]string, maxAllowedOrigin+1)); err == nil {
		t.Fatal("an oversized allowlist was accepted")
	}
}
