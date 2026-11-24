package middleware

import (
	"context"
	"errors"
	"math"
	"strings"

	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

const minRateLimitTTLMs int64 = 1000

// tokenBucketScript implementa um Token Bucket atomico no Redis usando o relogio do servidor (TIME).
// KEYS[1]: chave do bucket
// ARGV[1]: taxa de reposicao em tokens/segundo
// ARGV[2]: burst maximo (capacidade do bucket)
// ARGV[3]: TTL em milissegundos para expiracao automatica da chave
var tokenBucketScript = goredis.NewScript(`
local key = KEYS[1]
local refill_rate = tonumber(ARGV[1])
local burst = tonumber(ARGV[2])
local ttl_ms = tonumber(ARGV[3])

local now_tbl = redis.call("TIME")
local now_ms = tonumber(now_tbl[1]) * 1000 + math.floor(tonumber(now_tbl[2]) / 1000)

local data = redis.call("HMGET", key, "tokens", "ts")
local tokens = tonumber(data[1])
local last_ts = tonumber(data[2])

if tokens == nil or last_ts == nil then
	tokens = burst
	last_ts = now_ms
else
	local elapsed_ms = now_ms - last_ts
	if elapsed_ms > 0 then
		local refill = (elapsed_ms / 1000.0) * refill_rate
		tokens = math.min(burst, tokens + refill)
		last_ts = now_ms
	end
end

local allowed = 0
if tokens >= 1 then
	tokens = tokens - 1
	allowed = 1
end

redis.call("HSET", key, "tokens", tostring(tokens), "ts", tostring(last_ts))
redis.call("PEXPIRE", key, ttl_ms)

return allowed
`)

type redisRateLimiter struct {
	client goredis.Scripter
	prefix string
	r      rate.Limit
	b      int
}

func (l *redisRateLimiter) allow(ctx context.Context, rawKey string) (bool, error) {
	if l.b <= 0 || l.r <= 0 {
		return false, nil
	}
	if l.r == rate.Inf {
		return true, nil
	}
	if l.client == nil {
		return false, errors.New("cliente Redis nao inicializado")
	}

	redisKey := l.prefix + ":" + rawKey
	ttlMs := int64(math.Ceil(float64(l.b)/float64(l.r)*1000.0)) + minRateLimitTTLMs
	if ttlMs < minRateLimitTTLMs {
		ttlMs = minRateLimitTTLMs
	}

	res, err := tokenBucketScript.Run(ctx, l.client, []string{redisKey}, float64(l.r), l.b, ttlMs).Int()
	if err != nil {
		return false, err
	}

	return res == 1, nil
}

type EmailRateLimiter struct {
	limiter redisRateLimiter
}

func NewEmailRateLimiter(client goredis.Scripter, prefix string, r rate.Limit, b int) *EmailRateLimiter {
	return &EmailRateLimiter{
		limiter: redisRateLimiter{
			client: client,
			prefix: strings.TrimSpace(prefix),
			r:      r,
			b:      b,
		},
	}
}

func (e *EmailRateLimiter) Allow(ctx context.Context, email string) (bool, error) {
	key := strings.ToLower(strings.TrimSpace(email))
	return e.limiter.allow(ctx, key)
}

type IPRateLimiter struct {
	limiter redisRateLimiter
}

func NewRateLimiter(client goredis.Scripter, prefix string, r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		limiter: redisRateLimiter{
			client: client,
			prefix: strings.TrimSpace(prefix),
			r:      r,
			b:      b,
		},
	}
}

func (i *IPRateLimiter) Allow(ctx context.Context, ip string) (bool, error) {
	key := strings.TrimSpace(ip)
	return i.limiter.allow(ctx, key)
}
