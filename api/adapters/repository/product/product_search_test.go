package repository

import "testing"

func TestEscapeLike(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"sem especiais", "camisa", "camisa"},
		{"percent", "100%", `100\%`},
		{"underscore", "a_b", `a\_b`},
		{"barra invertida", `a\b`, `a\\b`},
		{"todos juntos", `%_\`, `\%\_\\`},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := escapeLike(c.in); got != c.want {
				t.Errorf("escapeLike(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}
