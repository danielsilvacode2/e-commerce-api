package brasilapi

import (
	"context"
	redisAdapter "e-commerce/adapters/redis"
	"e-commerce/core/apperrors"
	"e-commerce/core/ports"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultBaseURL        = "https://brasilapi.com.br"
	defaultRequestTimeout = 5 * time.Second
	defaultMaxRetries     = 2
	defaultCacheTTL       = 24 * time.Hour
	maxResponseBytes      = 4 << 20
)

type brasilAPIClient struct {
	baseURL        string
	httpClient     *http.Client
	cache          *redisAdapter.AddressCache
	requestTimeout time.Duration
	maxRetries     int
	retryDelays    []time.Duration
	cacheTTL       time.Duration
}

type cepV2Response struct {
	Cep          string `json:"cep"`
	State        string `json:"state"`
	City         string `json:"city"`
	Neighborhood string `json:"neighborhood"`
	Street       string `json:"street"`
}

type ibgeMunicipalityResponse struct {
	Nome string `json:"nome"`
}

func NewBrasilAPIClient() ports.AddressLookupGateway {
	return newBrasilAPIClientWithBaseURL(defaultBaseURL, redisAdapter.NewAddressCache(true))
}

func newBrasilAPIClientWithBaseURL(baseURL string, cache *redisAdapter.AddressCache) *brasilAPIClient {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	return &brasilAPIClient{
		baseURL:        baseURL,
		httpClient:     &http.Client{},
		cache:          cache,
		requestTimeout: defaultRequestTimeout,
		maxRetries:     defaultMaxRetries,
		retryDelays:    []time.Duration{200 * time.Millisecond, 500 * time.Millisecond},
		cacheTTL:       defaultCacheTTL,
	}
}

func (c *brasilAPIClient) LookupByCep(ctx context.Context, cep string) (*ports.CepLookupResult, error) {
	cleanCep := strings.ReplaceAll(strings.TrimSpace(cep), "-", "")
	if cleanCep == "" {
		return nil, apperrors.ErrInvalidInput
	}

	var cached ports.CepLookupResult
	if c.cache != nil && c.cache.GetCep(ctx, cleanCep, &cached) {
		return &cached, nil
	}

	endpoint := c.baseURL + "/api/cep/v2/" + url.PathEscape(cleanCep)
	statusCode, body, err := c.getWithRetry(ctx, endpoint)
	if err != nil {
		log.Printf("[ERROR] falha ao consultar CEP %s na BrasilAPI: %v", cleanCep, err)
		return nil, fmt.Errorf("%w: %v", apperrors.ErrAddressLookupUnavailable, err)
	}

	switch {
	case statusCode == http.StatusNotFound:
		return nil, apperrors.ErrCepNotFound
	case statusCode == http.StatusBadRequest:
		return nil, apperrors.ErrInvalidInput
	case statusCode < 200 || statusCode > 299:
		log.Printf("[ERROR] BrasilAPI CEP retornou status=%d para %s", statusCode, cleanCep)
		return nil, fmt.Errorf("%w: status %d", apperrors.ErrAddressLookupUnavailable, statusCode)
	}

	var payload cepV2Response
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[ERROR] falha ao decodificar resposta de CEP da BrasilAPI: %v", err)
		return nil, fmt.Errorf("%w: resposta invalida", apperrors.ErrAddressLookupUnavailable)
	}

	normalizedCep := strings.ReplaceAll(strings.TrimSpace(payload.Cep), "-", "")
	if normalizedCep == "" {
		normalizedCep = cleanCep
	}

	result := &ports.CepLookupResult{
		Cep:          normalizedCep,
		State:        strings.ToUpper(strings.TrimSpace(payload.State)),
		City:         strings.TrimSpace(payload.City),
		Neighborhood: strings.TrimSpace(payload.Neighborhood),
		Street:       strings.TrimSpace(payload.Street),
	}

	if result.State == "" || result.City == "" {
		log.Printf("[ERROR] BrasilAPI retornou CEP %s incompleto: %+v", cleanCep, result)
		return nil, fmt.Errorf("%w: resposta incompleta", apperrors.ErrAddressLookupUnavailable)
	}

	if c.cache != nil {
		c.cache.SetCep(ctx, cleanCep, result, c.cacheTTL)
	}

	return result, nil
}

func (c *brasilAPIClient) ListCitiesByState(ctx context.Context, uf string) ([]string, error) {
	cleanUF := strings.ToUpper(strings.TrimSpace(uf))
	if len(cleanUF) != 2 {
		return nil, apperrors.ErrInvalidInput
	}

	var cached []string
	if c.cache != nil && c.cache.GetCitiesByUF(ctx, cleanUF, &cached) && len(cached) > 0 {
		return cached, nil
	}

	endpoint := c.baseURL + "/api/ibge/municipios/v1/" + url.PathEscape(cleanUF)
	statusCode, body, err := c.getWithRetry(ctx, endpoint)
	if err != nil {
		log.Printf("[ERROR] falha ao consultar municipios da UF %s na BrasilAPI: %v", cleanUF, err)
		return nil, fmt.Errorf("%w: %v", apperrors.ErrAddressLookupUnavailable, err)
	}

	switch {
	case statusCode == http.StatusNotFound || statusCode == http.StatusBadRequest:
		return nil, apperrors.ErrInvalidInput
	case statusCode < 200 || statusCode > 299:
		log.Printf("[ERROR] BrasilAPI IBGE retornou status=%d para UF %s", statusCode, cleanUF)
		return nil, fmt.Errorf("%w: status %d", apperrors.ErrAddressLookupUnavailable, statusCode)
	}

	var payload []ibgeMunicipalityResponse
	if err := json.Unmarshal(body, &payload); err != nil {
		log.Printf("[ERROR] falha ao decodificar municipios da BrasilAPI para UF %s: %v", cleanUF, err)
		return nil, fmt.Errorf("%w: resposta invalida", apperrors.ErrAddressLookupUnavailable)
	}

	cities := make([]string, 0, len(payload))
	for _, item := range payload {
		name := strings.TrimSpace(item.Nome)
		if name != "" {
			cities = append(cities, name)
		}
	}

	if len(cities) == 0 {
		return nil, fmt.Errorf("%w: lista de municipios vazia", apperrors.ErrAddressLookupUnavailable)
	}

	if c.cache != nil {
		c.cache.SetCitiesByUF(ctx, cleanUF, cities, c.cacheTTL)
	}

	return cities, nil
}

func (c *brasilAPIClient) getWithRetry(ctx context.Context, endpoint string) (int, []byte, error) {
	attempts := c.maxRetries + 1
	if attempts < 1 {
		attempts = 1
	}

	var lastErr error
	var lastStatus int
	var lastBody []byte

	for attempt := 0; attempt < attempts; attempt++ {
		if ctx.Err() != nil {
			return 0, nil, ctx.Err()
		}

		reqCtx, cancel := context.WithTimeout(ctx, c.requestTimeout)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, endpoint, nil)
		if err != nil {
			cancel()
			return 0, nil, err
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "NERO-E-Commerce-API/1.0")

		res, err := c.httpClient.Do(req)
		if err != nil {
			cancel()
			lastErr = err
			if errors.Is(ctx.Err(), context.Canceled) {
				return 0, nil, err
			}
			if !c.sleepBeforeRetry(ctx, attempt, attempts) {
				return 0, nil, lastErr
			}
			continue
		}

		respBytes, readErr := io.ReadAll(io.LimitReader(res.Body, maxResponseBytes))
		res.Body.Close()
		cancel()

		if readErr != nil {
			lastErr = readErr
			if !c.sleepBeforeRetry(ctx, attempt, attempts) {
				return 0, nil, lastErr
			}
			continue
		}

		lastStatus = res.StatusCode
		lastBody = respBytes

		// Retry apenas em 429 ou erros transitorios 5xx
		if res.StatusCode == http.StatusTooManyRequests || res.StatusCode >= 500 {
			lastErr = fmt.Errorf("upstream status %d", res.StatusCode)
			if !c.sleepBeforeRetry(ctx, attempt, attempts) {
				return lastStatus, lastBody, nil
			}
			continue
		}

		return res.StatusCode, respBytes, nil
	}

	if lastStatus != 0 {
		return lastStatus, lastBody, nil
	}
	return 0, nil, lastErr
}

func (c *brasilAPIClient) sleepBeforeRetry(ctx context.Context, attempt int, totalAttempts int) bool {
	if attempt >= totalAttempts-1 {
		return false
	}

	delay := time.Duration(0)
	if attempt < len(c.retryDelays) {
		delay = c.retryDelays[attempt]
	} else if len(c.retryDelays) > 0 {
		delay = c.retryDelays[len(c.retryDelays)-1]
	}

	if delay <= 0 {
		return ctx.Err() == nil
	}

	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return false
	case <-timer.C:
		return true
	}
}
