package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

const (
	productCacheVersionKey = "cache:products:version"
	cacheOpTimeout         = 400 * time.Millisecond
)

func getProductCacheVersion(ctx context.Context) string {
	if Client == nil {
		return "0"
	}
	opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	ver, err := Client.Get(opCtx, productCacheVersionKey).Result()
	if err != nil || ver == "" {
		return "0"
	}
	return ver
}

// InvalidateProductCache incrementa a versao do cache de produtos em O(1), invalidando todas as paginas e detalhes em cache.
func InvalidateProductCache(ctx context.Context) {
	if Client == nil {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	_ = Client.Incr(opCtx, productCacheVersionKey).Err()
}

func productListCacheKey(ctx context.Context, page, limit int) string {
	return fmt.Sprintf("cache:products:v%s:list:p%d:l%d", getProductCacheVersion(ctx), page, limit)
}

func productByIDCacheKey(ctx context.Context, id string) string {
	return fmt.Sprintf("cache:products:v%s:id:%s", getProductCacheVersion(ctx), id)
}

// GetCachedProductList tenta ler a lista paginada de produtos do Redis.
func GetCachedProductList(ctx context.Context, page, limit int, dest any) bool {
	if Client == nil {
		return false
	}
	opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	raw, err := Client.Get(opCtx, productListCacheKey(opCtx, page, limit)).Bytes()
	if err != nil || len(raw) == 0 {
		return false
	}
	return json.Unmarshal(raw, dest) == nil
}

// SetCachedProductList grava a lista paginada de produtos no Redis com TTL curto.
func SetCachedProductList(ctx context.Context, page, limit int, value any, ttl time.Duration) {
	if Client == nil {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	_ = Client.Set(opCtx, productListCacheKey(opCtx, page, limit), raw, ttl).Err()
}

// GetCachedProductByID tenta ler o detalhe de um produto do Redis.
func GetCachedProductByID(ctx context.Context, id string, dest any) bool {
	if Client == nil || id == "" {
		return false
	}
	opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	raw, err := Client.Get(opCtx, productByIDCacheKey(opCtx, id)).Bytes()
	if err != nil || len(raw) == 0 {
		return false
	}
	return json.Unmarshal(raw, dest) == nil
}

// SetCachedProductByID grava o detalhe de um produto no Redis com TTL curto.
func SetCachedProductByID(ctx context.Context, id string, value any, ttl time.Duration) {
	if Client == nil || id == "" {
		return
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	_ = Client.Set(opCtx, productByIDCacheKey(opCtx, id), raw, ttl).Err()
}

// AcquirePaymentGatewaySyncSlot garante cooldown distribuido antes de consultar o status no gateway externo.
// Retorna true quando a consulta ao gateway esta liberada (ou quando o Redis nao esta inicializado).
func AcquirePaymentGatewaySyncSlot(ctx context.Context, externalPaymentID string, cooldown time.Duration) bool {
	if Client == nil || externalPaymentID == "" {
		return true
	}
	opCtx, cancel := context.WithTimeout(ctx, cacheOpTimeout)
	defer cancel()

	ok, err := Client.SetNX(opCtx, "cooldown:payment_gw:"+externalPaymentID, "1", cooldown).Result()
	if err != nil {
		return true
	}
	return ok
}
