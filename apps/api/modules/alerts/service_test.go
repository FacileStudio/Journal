package alerts

import "testing"

func TestValidateRule(t *testing.T) {
	cases := []struct {
		name          string
		ruleName      string
		threshold     int
		windowMinutes int
		webhookURL    string
		valid         bool
	}{
		{"valid https", "errors spike", 5, 15, "https://nook.example.com/hooks/abc", true},
		{"valid http", "errors spike", 1, 1, "http://localhost:9000/hook", true},
		{"max window", "errors spike", 1, 1440, "https://example.com/h", true},
		{"empty name", "", 5, 15, "https://example.com/h", false},
		{"whitespace name", "   ", 5, 15, "https://example.com/h", false},
		{"zero threshold", "a", 0, 15, "https://example.com/h", false},
		{"negative threshold", "a", -1, 15, "https://example.com/h", false},
		{"zero window", "a", 1, 0, "https://example.com/h", false},
		{"window too large", "a", 1, 1441, "https://example.com/h", false},
		{"empty url", "a", 1, 15, "", false},
		{"no scheme", "a", 1, 15, "example.com/h", false},
		{"bad scheme", "a", 1, 15, "ftp://example.com/h", false},
		{"no host", "a", 1, 15, "https:///path", false},
		{"unparseable url", "a", 1, 15, "http://[::1", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateRule(tc.ruleName, tc.threshold, tc.windowMinutes, tc.webhookURL)
			if tc.valid && err != nil {
				t.Fatalf("validateRule = %v, want nil", err)
			}
			if !tc.valid && err == nil {
				t.Fatal("validateRule = nil, want error")
			}
		})
	}
}
