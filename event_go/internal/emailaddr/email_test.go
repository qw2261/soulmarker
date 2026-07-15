package emailaddr

import "testing"

func TestNormalize(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{input: " User@Example.COM ", want: "user@example.com", ok: true},
		{input: "User <user@example.com>", ok: false},
		{input: "13800138000", ok: false},
		{input: "", ok: false},
	}
	for _, tt := range tests {
		got, ok := Normalize(tt.input)
		if got != tt.want || ok != tt.ok {
			t.Fatalf("Normalize(%q) = %q, %v; want %q, %v", tt.input, got, ok, tt.want, tt.ok)
		}
	}
}
