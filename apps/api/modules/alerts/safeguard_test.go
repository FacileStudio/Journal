package alerts

import (
	"context"
	stderrors "errors"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsBlockedIP(t *testing.T) {
	cases := []struct {
		name    string
		ip      string
		blocked bool
	}{
		{"loopback v4", "127.0.0.1", true},
		{"loopback v6", "::1", true},
		{"metadata", "169.254.169.254", true},
		{"private 10", "10.0.0.5", true},
		{"private 172", "172.16.0.1", true},
		{"private 192", "192.168.1.1", true},
		{"ula v6", "fc00::1", true},
		{"multicast", "224.0.0.1", true},
		{"unspecified", "0.0.0.0", true},
		{"public google", "8.8.8.8", false},
		{"public cloudflare", "1.1.1.1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ip := net.ParseIP(tc.ip)
			if ip == nil {
				t.Fatalf("ParseIP(%q) = nil", tc.ip)
			}
			if got := isBlockedIP(ip); got != tc.blocked {
				t.Fatalf("isBlockedIP(%s) = %v, want %v", tc.ip, got, tc.blocked)
			}
		})
	}
}

func TestHostAllowed(t *testing.T) {
	allowed := []string{"nook", "Hooks.Internal"}
	cases := []struct {
		host string
		want bool
	}{
		{"nook", true},
		{"NOOK", true},
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
	} else if !stderrors.Is(err, errBlockedWebhookTarget) {
		t.Logf("guarded dial rejected as expected: %v", err)
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
