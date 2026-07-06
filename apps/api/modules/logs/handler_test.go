package logs

import "testing"

func TestPickBucketSeconds(t *testing.T) {
	cases := []struct {
		name         string
		rangeSeconds int64
		want         int64
	}{
		{"one hour", 3600, 60},
		{"ninety minutes boundary", 5400, 60},
		{"two hours", 7200, 300},
		{"six hours", 21600, 300},
		{"twelve hours", 43200, 900},
		{"one day", 86400, 3600},
		{"three days", 259200, 3600},
		{"seven days", 604800, 21600},
		{"thirty days", 2592000, 86400},
		{"ninety days boundary", 7776000, 86400},
		{"one year falls back", 31536000, 86400},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := pickBucketSeconds(tc.rangeSeconds); got != tc.want {
				t.Fatalf("pickBucketSeconds(%d) = %d, want %d", tc.rangeSeconds, got, tc.want)
			}
		})
	}
}
