package dto

import (
	"strings"
	"testing"
)

func TestProductSearchRequestDTOValidator(t *testing.T) {
	cases := []struct {
		name      string
		query     string
		wantErr   bool
		wantQuery string
	}{
		{"valido", "camisa", false, "camisa"},
		{"trim", "  camisa  ", false, "camisa"},
		{"vazio", "", true, ""},
		{"somente espacos", "    ", true, ""},
		{"curto", "a", true, "a"},
		{"minimo", "ab", false, "ab"},
		{"caracteres especiais", "100%_", false, "100%_"},
		{"maximo", strings.Repeat("a", 100), false, strings.Repeat("a", 100)},
		{"acima do maximo", strings.Repeat("a", 101), true, strings.Repeat("a", 101)},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			dto := ProductSearchRequestDTO{Query: c.query}
			errs := dto.Validator()

			if (len(errs) > 0) != c.wantErr {
				t.Fatalf("erros = %v, wantErr %v", errs, c.wantErr)
			}
			if dto.Query != c.wantQuery {
				t.Errorf("query = %q, want %q", dto.Query, c.wantQuery)
			}
		})
	}
}
