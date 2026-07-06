package apikeys

import (
	"strings"
	"testing"

	"github.com/FacileStudio/Journal/apps/api/internal/authcrypto"
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
	token, prefix, hash, err := generateToken("nuage")
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

	other, _, otherHash, err := generateToken("nuage")
	if err != nil {
		t.Fatalf("generateToken: %v", err)
	}
	if token == other || hash == otherHash {
		t.Fatal("two generated tokens collide")
	}
}
