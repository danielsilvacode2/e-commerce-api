package ports

import "context"

// CepLookupResult representa os dados de endereco retornados por um provedor externo de CEP.
type CepLookupResult struct {
	Cep          string
	State        string
	City         string
	Neighborhood string
	Street       string
}

// AddressLookupGateway define a porta para consulta de CEP e listagem de municipios por UF.
type AddressLookupGateway interface {
	LookupByCep(ctx context.Context, cep string) (*CepLookupResult, error)
	ListCitiesByState(ctx context.Context, uf string) ([]string, error)
}
