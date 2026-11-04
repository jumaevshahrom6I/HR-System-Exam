package employee

import "testing"

func TestEscapeLike(t *testing.T) {
	tests := map[string]string{
		"ivan":    "ivan",
		"50%":     `50\%`,
		"a_b":     `a\_b`,
		`back\sl`: `back\\sl`,
		`%_\`:     `\%\_\\`,
		"":        "",
	}
	for in, want := range tests {
		if got := escapeLike(in); got != want {
			t.Fatalf("escapeLike(%q) = %q, want %q", in, got, want)
		}
	}
}
