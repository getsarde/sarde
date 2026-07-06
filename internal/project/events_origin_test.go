package project

import "testing"

func TestAllowedEventOrigin(t *testing.T) {
	cases := []struct {
		origin string
		want   bool
	}{
		{"", true}, // non-browser client; token middleware already vetted it
		{"tauri://localhost", true},
		{"https://tauri.localhost", true},
		{"http://localhost:5173", true},
		{"http://127.0.0.1:4727", true},
		{"https://evil.example.com", false},
		{"http://attacker.localhost.evil.com", false},
		{"file://x", false},
	}
	for _, tc := range cases {
		if got := allowedEventOrigin(tc.origin); got != tc.want {
			t.Errorf("allowedEventOrigin(%q) = %v, want %v", tc.origin, got, tc.want)
		}
	}
}
