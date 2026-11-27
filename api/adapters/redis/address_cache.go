package redis

import (
	"context"
	"encoding/json"
	"sync"
	"time"
)

type memoryCacheEntry struct {
	payload   []byte
	expiresAt time.Time
}

// AddressCache fornece cache com TTL para consultas de CEP e municipios por UF,
// utilizando Redis quando disponivel com fallback automatico em memoria.
type AddressCache struct {
	useRedis bool
	mu       sync.RWMutex
	entries  map[string]memoryCacheEntry
}

func NewAddressCache(useRedis bool) *AddressCache {
	return &AddressCache{
		useRedis: useRedis,
		entries:  make(map[string]memoryCacheEntry),
	}
}

func cepCacheKey(cep string) string {
	return "cache:address:cep:" + cep
}

func ufCitiesCacheKey(uf string) string {
	return "cache:address:ibge:uf:" + uf
}

func (c *AddressCache) GetCep(ctx context.Context, cep string, dest any) bool {
	if c == nil || cep == "" {
		return false
	}
	return c.get(ctx, cepCacheKey(cep), dest)
}

func (c *AddressCache) SetCep(ctx context.Context, cep string, value any, ttl time.Duration) {
	if c == nil || cep == "" {
		return
	}
	c.set(ctx, cepCacheKey(cep), value, ttl)
}

func (c *AddressCache) GetCitiesByUF(ctx context.Context, uf string, dest any) bool {
	if c == nil || uf == "" {
		return false
	}
	return c.get(ctx, ufCitiesCacheKey(uf), dest)
}

func (c *AddressCache) SetCitiesByUF(ctx context.Context, uf string, value any, ttl time.Duration) {
	if c == nil || uf == "" {
		return
	}
	c.set(ctx, ufCitiesCacheKey(uf), value, ttl)
}

func (c *AddressCache) get(ctx context.Context, key string, dest any) bool {
	if c.useRedis && Client != nil {
		opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
		defer cancel()

		raw, err := Client.Get(opCtx, key).Bytes()
		if err == nil && len(raw) > 0 {
			if json.Unmarshal(raw, dest) == nil {
				return true
			}
		}
	}

	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return false
	}

	return json.Unmarshal(entry.payload, dest) == nil
}

func (c *AddressCache) set(ctx context.Context, key string, value any, ttl time.Duration) {
	if ttl <= 0 {
		return
	}

	raw, err := json.Marshal(value)
	if err != nil {
		return
	}

	if c.useRedis && Client != nil {
		opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
		defer cancel()
		_ = Client.Set(opCtx, key, raw, ttl).Err()
	}

	c.mu.Lock()
	c.entries[key] = memoryCacheEntry{
		payload:   raw,
		expiresAt: time.Now().Add(ttl),
	}
	c.mu.Unlock()
}
