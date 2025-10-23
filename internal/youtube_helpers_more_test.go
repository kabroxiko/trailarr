package internal

import "testing"

func TestIsImpersonationErrorNative(t *testing.T) {
	cases := []struct {
		out  string
		want bool
	}{
		{"Impersonate target not found", true},
		{"missing dependencies required to support this target", true},
		{"all good", false},
	}
	for _, c := range cases {
		if got := isImpersonationErrorNative(c.out); got != c.want {
			t.Fatalf("isImpersonationErrorNative(%q) = %v, want %v", c.out, got, c.want)
		}
	}
}
