package alerts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/FacileStudio/tronc/netguard"
)

func TestValidateRuleRefusesInternalLiterals(t *testing.T) {
	cases := []struct {
		name    string
		host    string
		blocked bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "[::1]", true},
		{"metadata", "169.254.169.254", true},
		{"private 10", "10.0.0.5", true},
		{"private 172", "172.16.0.1", true},
		{"private 192", "192.168.1.1", true},
		{"ula v6", "[fc00::1]", true},
		{"multicast", "224.0.0.1", true},
		{"unspecified", "0.0.0.0", true},
		{"cgnat", "100.64.0.1", true},
		{"cgnat alibaba metadata", "100.100.100.200", true},
		{"reserved 240/4", "240.0.0.1", true},
		{"this network 0/8", "0.1.2.3", true},
		{"nat64 of metadata", "[64:ff9b::a9fe:a9fe]", true},
		{"6to4 of loopback", "[2002:7f00:1::1]", true},
		{"mapped loopback", "[::ffff:127.0.0.1]", true},
		{"public google", "8.8.8.8", false},
		{"public cloudflare", "1.1.1.1", false},
		{"nat64 of public", "[64:ff9b::808:808]", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRule("rule", 1, 5, "https://"+tc.host+"/hook", "webhook")
			if (err != nil) != tc.blocked {
				t.Fatalf("validateRule(%s) error = %v, want blocked=%v", tc.host, err, tc.blocked)
			}
		})
	}
}

func TestHostAllowed(t *testing.T) {
	allowed := []string{"antenne", "Hooks.Internal"}
	cases := []struct {
		host string
		want bool
	}{
		{"antenne", true},
		{"ANTENNE", true},
		{"hooks.internal", true},
		{"other", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := hostAllowed(tc.host, allowed); got != tc.want {
			t.Fatalf("hostAllowed(%q) = %v, want %v", tc.host, got, tc.want)
		}
	}
}

func TestGuardedClientRefusesLoopback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	guarded := guardedClient(2 * time.Second)
	request, err := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	if _, err := guarded.Do(request); err == nil {
		t.Fatal("guarded client reached loopback server, want dial error")
	} else if _, ok := netguard.AsBlocked(err); !ok {
		t.Fatalf("guarded client error = %v, want a netguard refusal", err)
	}

	trusted := trustedClient(2 * time.Second)
	request, err = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL, nil)
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	response, err := trusted.Do(request)
	if err != nil {
		t.Fatalf("trusted client failed to reach loopback: %v", err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		t.Fatalf("trusted status = %d, want 200", response.StatusCode)
	}
}
