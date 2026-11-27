package entities

import (
	"e-commerce/core/apperrors"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
	"golang.org/x/text/unicode/norm"
)

const (
	MaxStreetLength       = 200
	MaxNumberLength       = 30
	MaxComplementLength   = 120
	MaxNeighborhoodLength = 120
	MaxCityLength         = 120
)

var cepPattern = regexp.MustCompile(`^[0-9]{5}-?[0-9]{3}$`)

var validBrazilianUFs = map[string]struct{}{
	"AC": {}, "AL": {}, "AP": {}, "AM": {}, "BA": {}, "CE": {}, "DF": {},
	"ES": {}, "GO": {}, "MA": {}, "MT": {}, "MS": {}, "MG": {}, "PA": {},
	"PB": {}, "PR": {}, "PE": {}, "PI": {}, "RJ": {}, "RN": {}, "RS": {},
	"RO": {}, "RR": {}, "SC": {}, "SP": {}, "SE": {}, "TO": {},
}

type Address struct {
	Id           uuid.UUID
	UserId       uuid.UUID
	Cep          string
	Street       string
	Number       string
	Complement   string
	Neighborhood string
	City         string
	State        string
	UpdateAt     time.Time
	CreateAt     time.Time
}

// BelongsTo verifica se o endereco pertence ao usuario informado.
func (a *Address) BelongsTo(userId string) bool {
	return a != nil && a.UserId != uuid.Nil && a.UserId.String() == userId
}

// IsComplete indica se o endereco possui todos os campos obrigatorios preenchidos
// (diferenciando enderecos completos de registros legados migrados apenas com CEP).
func (a *Address) IsComplete() bool {
	if a == nil {
		return false
	}
	_, errs := a.ValidateLocal()
	return len(errs) == 0
}

type AddressRepository interface {
	CreateAddress(address Address) error
	ReadAddressById(ID string) (*Address, error)
	ReadAddressesByUserId(userId string) ([]Address, error)
	FindMatchingUserAddress(userId string, address Address) (*Address, error)
}

// NormalizeCep valida o formato (00000-000 ou 00000000) e retorna apenas os 8 digitos.
func NormalizeCep(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if !cepPattern.MatchString(trimmed) {
		return "", apperrors.NewValidationError(
			apperrors.ErrInvalidInput,
			"CEP inválido. Informe 8 dígitos no formato 00000-000 ou 00000000.",
			apperrors.FieldError{
				Field:   "cep",
				Message: "Informe um CEP válido com 8 dígitos (ex.: 00000-000).",
			},
		)
	}
	return strings.ReplaceAll(trimmed, "-", ""), nil
}

// FormatCep formata um CEP de 8 digitos como 00000-000.
func FormatCep(cep string) string {
	digits := strings.ReplaceAll(strings.TrimSpace(cep), "-", "")
	if len(digits) != 8 {
		return strings.TrimSpace(cep)
	}
	return digits[:5] + "-" + digits[5:]
}

// NormalizeState retorna a sigla UF em maiusculas e sem espacos laterais.
func NormalizeState(raw string) string {
	return strings.ToUpper(strings.TrimSpace(raw))
}

// IsValidUF verifica se a sigla corresponde a uma das 27 UFs brasileiras.
func IsValidUF(uf string) bool {
	_, ok := validBrazilianUFs[NormalizeState(uf)]
	return ok
}

// NormalizeCityName normaliza nomes de cidades para comparacao insensivel a acentos, caixa e espacos extras.
func NormalizeCityName(raw string) string {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return ""
	}

	decomposed := norm.NFD.String(trimmed)
	var b strings.Builder
	b.Grow(len(decomposed))

	lastSpace := false
	for _, r := range decomposed {
		if unicode.Is(unicode.Mn, r) {
			continue
		}
		if unicode.IsSpace(r) {
			if !lastSpace {
				b.WriteByte(' ')
				lastSpace = true
			}
			continue
		}
		b.WriteRune(unicode.ToLower(r))
		lastSpace = false
	}

	return strings.TrimSpace(b.String())
}

// MatchesCity compara duas cidades ignorando acentos, caixa e espacos excedentes.
func MatchesCity(a, b string) bool {
	na := NormalizeCityName(a)
	nb := NormalizeCityName(b)
	return na != "" && na == nb
}

// Normalize retorna uma copia do endereco com os campos sanitizados (mantendo CEP em 8 digitos quando valido).
func (a Address) Normalize() Address {
	normalized := Address{
		Id:           a.Id,
		UserId:       a.UserId,
		Cep:          strings.TrimSpace(a.Cep),
		Street:       strings.TrimSpace(a.Street),
		Number:       strings.TrimSpace(a.Number),
		Complement:   strings.TrimSpace(a.Complement),
		Neighborhood: strings.TrimSpace(a.Neighborhood),
		City:         strings.TrimSpace(a.City),
		State:        NormalizeState(a.State),
		UpdateAt:     a.UpdateAt,
		CreateAt:     a.CreateAt,
	}
	if digits, err := NormalizeCep(normalized.Cep); err == nil {
		normalized.Cep = digits
	}
	return normalized
}

// ValidateLocal executa as validacoes puras de dominio sem depender de servicos externos.
func (a Address) ValidateLocal() (Address, []apperrors.FieldError) {
	n := a.Normalize()
	var errs []apperrors.FieldError

	if _, err := NormalizeCep(a.Cep); err != nil {
		errs = append(errs, apperrors.FieldError{
			Field:   "cep",
			Message: "Informe um CEP válido com 8 dígitos (ex.: 00000-000).",
		})
	}

	if n.Street == "" {
		errs = append(errs, apperrors.FieldError{
			Field:   "street",
			Message: "A rua é obrigatória.",
		})
	} else if len([]rune(n.Street)) > MaxStreetLength {
		errs = append(errs, apperrors.FieldError{
			Field:   "street",
			Message: "A rua deve ter no máximo 200 caracteres.",
		})
	}

	if n.Number == "" {
		errs = append(errs, apperrors.FieldError{
			Field:   "number",
			Message: "O número é obrigatório (informe S/N se não houver).",
		})
	} else if len([]rune(n.Number)) > MaxNumberLength {
		errs = append(errs, apperrors.FieldError{
			Field:   "number",
			Message: "O número deve ter no máximo 30 caracteres.",
		})
	}

	if len([]rune(n.Complement)) > MaxComplementLength {
		errs = append(errs, apperrors.FieldError{
			Field:   "complement",
			Message: "O complemento deve ter no máximo 120 caracteres.",
		})
	}

	if len([]rune(n.Neighborhood)) > MaxNeighborhoodLength {
		errs = append(errs, apperrors.FieldError{
			Field:   "neighborhood",
			Message: "O bairro deve ter no máximo 120 caracteres.",
		})
	}

	if n.City == "" {
		errs = append(errs, apperrors.FieldError{
			Field:   "city",
			Message: "A cidade é obrigatória.",
		})
	} else if len([]rune(n.City)) > MaxCityLength {
		errs = append(errs, apperrors.FieldError{
			Field:   "city",
			Message: "A cidade deve ter no máximo 120 caracteres.",
		})
	}

	if n.State == "" {
		errs = append(errs, apperrors.FieldError{
			Field:   "state",
			Message: "O estado (UF) é obrigatório.",
		})
	} else if !IsValidUF(n.State) {
		errs = append(errs, apperrors.FieldError{
			Field:   "state",
			Message: "Informe uma sigla de estado (UF) brasileira válida.",
		})
	}

	return n, errs
}
