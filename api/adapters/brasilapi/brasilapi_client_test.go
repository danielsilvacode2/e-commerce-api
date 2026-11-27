package brasilapi

import (
	"bytes"
	"context"
	redisAdapter "e-commerce/adapters/redis"
	"e-commerce/core/apperrors"
	"errors"
	"io"
	"net/http"
	"testing"
	"time"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(bytes.NewBufferString(body)),
	}
}

func newTestBrasilAPIClient(fn roundTripFunc) *brasilAPIClient {
	client := newBrasilAPIClientWithBaseURL("https://brasilapi.test", redisAdapter.NewAddressCache(false))
	client.httpClient = &http.Client{Transport: fn}
	client.retryDelays = []time.Duration{0, 0}
	return client
}

func TestLookupByCepSuccessAndCache(t *testing.T) {
	calls := 0
	client := newTestBrasilAPIClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/api/cep/v2/01001000" {
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
		return jsonResponse(http.StatusOK, `{"cep":"01001000","state":"SP","city":"São Paulo","neighborhood":"Sé","street":"Praça da Sé"}`), nil
	})

	res, err := client.LookupByCep(context.Background(), "01001-000")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if res.Cep != "01001000" || res.State != "SP" || res.City != "São Paulo" || res.Street != "Praça da Sé" || res.Neighborhood != "Sé" {
		t.Fatalf("resultado inesperado: %+v", res)
	}

	// Segunda chamada deve vir do cache sem acionar HTTP
	res2, err := client.LookupByCep(context.Background(), "01001000")
	if err != nil {
		t.Fatalf("erro inesperado na consulta em cache: %v", err)
	}
	if calls != 1 {
		t.Fatalf("esperava 1 chamada HTTP com cache, obteve %d", calls)
	}
	if res2.City != "São Paulo" {
		t.Fatalf("cache retornou cidade inesperada: %q", res2.City)
	}
}

func TestLookupByCepNotFoundNoRetry(t *testing.T) {
	calls := 0
	client := newTestBrasilAPIClient(func(r *http.Request) (*http.Response, error) {
		calls++
		return jsonResponse(http.StatusNotFound, `{"message":"Todos os serviços de CEP retornaram erro."}`), nil
	})

	_, err := client.LookupByCep(context.Background(), "99999999")
	if !errors.Is(err, apperrors.ErrCepNotFound) {
		t.Fatalf("esperava ErrCepNotFound, obteve %v", err)
	}
	if calls != 1 {
		t.Fatalf("nao deveria fazer retry em 404, obteve %d chamadas", calls)
	}
}

func TestLookupByCepRetryOn5xxAndSucceeds(t *testing.T) {
	calls := 0
	client := newTestBrasilAPIClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if calls < 3 {
			return jsonResponse(http.StatusBadGateway, `{"error":"upstream fail"}`), nil
		}
		return jsonResponse(http.StatusOK, `{"cep":"20040020","state":"RJ","city":"Rio de Janeiro","neighborhood":"Centro","street":"Rua da Assembleia"}`), nil
	})

	res, err := client.LookupByCep(context.Background(), "20040020")
	if err != nil {
		t.Fatalf("esperava sucesso na 3a tentativa, obteve %v", err)
	}
	if calls != 3 {
		t.Fatalf("esperava 3 tentativas, obteve %d", calls)
	}
	if res.State != "RJ" {
		t.Fatalf("estado inesperado: %s", res.State)
	}
}

func TestLookupByCepUnavailableAfterRetries(t *testing.T) {
	calls := 0
	client := newTestBrasilAPIClient(func(r *http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("connection timeout")
	})

	_, err := client.LookupByCep(context.Background(), "01001000")
	if !errors.Is(err, apperrors.ErrAddressLookupUnavailable) {
		t.Fatalf("esperava ErrAddressLookupUnavailable, obteve %v", err)
	}
	if calls != 3 {
		t.Fatalf("esperava 3 tentativas (1 + 2 retries), obteve %d", calls)
	}
}

func TestListCitiesByStateSuccessAndCache(t *testing.T) {
	calls := 0
	client := newTestBrasilAPIClient(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.Path != "/api/ibge/municipios/v1/SP" {
			t.Fatalf("path inesperado: %s", r.URL.Path)
		}
		return jsonResponse(http.StatusOK, `[{"nome":"SÃO PAULO","codigo_ibge":"3550308"},{"nome":"CAMPINAS","codigo_ibge":"3509502"}]`), nil
	})

	cities, err := client.ListCitiesByState(context.Background(), "sp")
	if err != nil {
		t.Fatalf("erro inesperado: %v", err)
	}
	if len(cities) != 2 || cities[0] != "SÃO PAULO" {
		t.Fatalf("municipios inesperados: %+v", cities)
	}

	// Segunda chamada via cache
	cities2, err := client.ListCitiesByState(context.Background(), "SP")
	if err != nil || len(cities2) != 2 || calls != 1 {
		t.Fatalf("esperava cache na 2a chamada: calls=%d err=%v cities=%+v", calls, err, cities2)
	}
}
