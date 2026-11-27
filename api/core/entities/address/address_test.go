package entities_test

import (
	entitiesAddress "e-commerce/core/entities/address"
	"testing"
)

func TestNormalizeCep(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{name: "cep sem mascara", input: "01001000", want: "01001000"},
		{name: "cep com mascara", input: "01001-000", want: "01001000"},
		{name: "cep com espacos", input: "  01001-000  ", want: "01001000"},
		{name: "cep curto", input: "0100100", wantErr: true},
		{name: "cep com letras", input: "01001-00A", wantErr: true},
		{name: "cep vazio", input: "", wantErr: true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := entitiesAddress.NormalizeCep(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("esperava erro para %q, obteve %q", tc.input, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("erro inesperado para %q: %v", tc.input, err)
			}
			if got != tc.want {
				t.Fatalf("NormalizeCep(%q) = %q, esperado %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestIsValidUF(t *testing.T) {
	for _, uf := range []string{"SP", "rj", " mg ", "DF", "RS", "BA", "TO"} {
		if !entitiesAddress.IsValidUF(uf) {
			t.Fatalf("esperava UF valida para %q", uf)
		}
	}
	for _, uf := range []string{"", "XX", "S", "SAO", "12"} {
		if entitiesAddress.IsValidUF(uf) {
			t.Fatalf("esperava UF invalida para %q", uf)
		}
	}
}

func TestMatchesCity(t *testing.T) {
	if !entitiesAddress.MatchesCity("São Paulo", "sao paulo") {
		t.Fatal("deveria ignorar acentos e caixa")
	}
	if !entitiesAddress.MatchesCity("  Florianópolis  ", "FLORIANOPOLIS") {
		t.Fatal("deveria ignorar espacos, acentos e caixa")
	}
	if !entitiesAddress.MatchesCity("Belém   do  Pará", "belem do para") {
		t.Fatal("deveria colapsar espacos internos")
	}
	if entitiesAddress.MatchesCity("São Paulo", "Campinas") {
		t.Fatal("cidades diferentes nao devem casar")
	}
}

func TestAddressValidateLocal(t *testing.T) {
	valid := entitiesAddress.Address{
		Cep:          "01001-000",
		Street:       "  Praça da Sé ",
		Number:       "123A",
		Complement:   " Bloco B ",
		Neighborhood: " Sé ",
		City:         "São Paulo",
		State:        "sp",
	}

	normalized, errs := valid.ValidateLocal()
	if len(errs) != 0 {
		t.Fatalf("nao esperava erros, obteve %+v", errs)
	}
	if normalized.Cep != "01001000" || normalized.State != "SP" || normalized.Street != "Praça da Sé" {
		t.Fatalf("normalizacao inesperada: %+v", normalized)
	}

	invalid := entitiesAddress.Address{
		Cep:    "123",
		Street: "   ",
		Number: "",
		City:   "",
		State:  "XX",
	}
	_, errs = invalid.ValidateLocal()
	if len(errs) != 5 {
		t.Fatalf("esperava 5 erros de campo, obteve %d (%+v)", len(errs), errs)
	}
}
