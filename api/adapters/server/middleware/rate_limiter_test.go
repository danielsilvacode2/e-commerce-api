package middleware

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	goredis "github.com/redis/go-redis/v9"
	"golang.org/x/time/rate"
)

type fakeRedisScripter struct {
	buckets  map[string]int
	err      error
	lastKey  string
	lastArgs []any
}

func newFakeRedisScripter() *fakeRedisScripter {
	return &fakeRedisScripter{
		buckets: make(map[string]int),
	}
}

func (f *fakeRedisScripter) evalBucket(keys []string, args ...any) *goredis.Cmd {
	if f.err != nil {
		return goredis.NewCmdResult(nil, f.err)
	}
	if len(keys) > 0 {
		f.lastKey = keys[0]
	}
	f.lastArgs = args

	burst, _ := args[1].(int)
	key := keys[0]
	remaining, exists := f.buckets[key]
	if !exists {
		remaining = burst
	}

	if remaining >= 1 {
		f.buckets[key] = remaining - 1
		return goredis.NewCmdResult(int64(1), nil)
	}

	return goredis.NewCmdResult(int64(0), nil)
}

func (f *fakeRedisScripter) Eval(_ context.Context, _ string, keys []string, args ...any) *goredis.Cmd {
	return f.evalBucket(keys, args...)
}

func (f *fakeRedisScripter) EvalSha(_ context.Context, _ string, keys []string, args ...any) *goredis.Cmd {
	return f.evalBucket(keys, args...)
}

func (f *fakeRedisScripter) EvalRO(_ context.Context, _ string, keys []string, args ...any) *goredis.Cmd {
	return f.evalBucket(keys, args...)
}

func (f *fakeRedisScripter) EvalShaRO(_ context.Context, _ string, keys []string, args ...any) *goredis.Cmd {
	return f.evalBucket(keys, args...)
}

func (f *fakeRedisScripter) ScriptExists(_ context.Context, _ ...string) *goredis.BoolSliceCmd {
	return goredis.NewBoolSliceResult([]bool{true}, nil)
}

func (f *fakeRedisScripter) ScriptLoad(_ context.Context, _ string) *goredis.StringCmd {
	return goredis.NewStringResult("fake-sha", nil)
}

func TestEmailRateLimiter_NormalizationAndPrefixIsolation(t *testing.T) {
	fake := newFakeRedisScripter()
	loginLimiter := NewEmailRateLimiter(fake, "rl:email:login", rate.Every(5*time.Minute), 2)
	verifyLimiter := NewEmailRateLimiter(fake, "rl:email:verify_code", rate.Every(20*time.Minute), 2)

	ctx := context.Background()

	allowed, err := loginLimiter.Allow(ctx, "  User@Example.COM ")
	if err != nil || !allowed {
		t.Fatalf("esperava primeira requisicao permitida, got allowed=%v err=%v", allowed, err)
	}
	if fake.lastKey != "rl:email:login:user@example.com" {
		t.Fatalf("chave Redis inesperada: %q", fake.lastKey)
	}

	allowed, err = loginLimiter.Allow(ctx, "user@example.com")
	if err != nil || !allowed {
		t.Fatalf("esperava segunda requisicao permitida, got allowed=%v err=%v", allowed, err)
	}

	allowed, err = loginLimiter.Allow(ctx, "USER@example.com")
	if err != nil || allowed {
		t.Fatalf("esperava terceira requisicao bloqueada, got allowed=%v err=%v", allowed, err)
	}

	// Outro limiter com prefixo diferente nao pode colidir para o mesmo email
	allowed, err = verifyLimiter.Allow(ctx, "user@example.com")
	if err != nil || !allowed {
		t.Fatalf("esperava verifyLimiter isolado do loginLimiter, got allowed=%v err=%v", allowed, err)
	}
	if fake.lastKey != "rl:email:verify_code:user@example.com" {
		t.Fatalf("chave Redis inesperada no verifyLimiter: %q", fake.lastKey)
	}
}

func TestIPRateLimiter_EdgeCases(t *testing.T) {
	ctx := context.Background()

	nilLimiter := NewRateLimiter(nil, "rl:ip:global", rate.Every(time.Second), 5)
	if allowed, err := nilLimiter.Allow(ctx, "127.0.0.1"); err == nil || allowed {
		t.Fatalf("esperava erro com cliente Redis nil, got allowed=%v err=%v", allowed, err)
	}

	var typedNilClient *goredis.Client
	typedNilLimiter := NewRateLimiter(typedNilClient, "rl:ip:global", rate.Every(time.Second), 5)
	if allowed, err := typedNilLimiter.Allow(ctx, "127.0.0.1"); err == nil || allowed {
		t.Fatalf("esperava erro com cliente Redis typed-nil sem panic, got allowed=%v err=%v", allowed, err)
	}

	fake := newFakeRedisScripter()
	zeroBurst := NewRateLimiter(fake, "rl:ip:global", rate.Every(time.Second), 0)
	if allowed, err := zeroBurst.Allow(ctx, "127.0.0.1"); err != nil || allowed {
		t.Fatalf("esperava bloqueio com burst 0, got allowed=%v err=%v", allowed, err)
	}

	infLimiter := NewRateLimiter(fake, "rl:ip:global", rate.Inf, 1)
	if allowed, err := infLimiter.Allow(ctx, "127.0.0.1"); err != nil || !allowed {
		t.Fatalf("esperava permissao com rate.Inf, got allowed=%v err=%v", allowed, err)
	}
}

func TestRateLimiterMiddleware_BlocksAfterBurstAndFailsOpenOnRedisError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	fake := newFakeRedisScripter()
	limiter := NewRateLimiter(fake, "rl:ip:global", rate.Every(time.Minute), 2)

	r := gin.New()
	r.Use(RateLimiterMiddleware(limiter))
	r.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/ping", nil)
		req.RemoteAddr = "192.168.0.10:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Fatalf("requisicao %d deveria retornar 200, retornou %d", i+1, w.Code)
		}
	}

	// Terceira requisicao do mesmo IP deve ser bloqueada com 429
	reqBlocked := httptest.NewRequest(http.MethodGet, "/ping", nil)
	reqBlocked.RemoteAddr = "192.168.0.10:12345"
	wBlocked := httptest.NewRecorder()
	r.ServeHTTP(wBlocked, reqBlocked)
	if wBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("esperava 429 apos exceder burst, retornou %d", wBlocked.Code)
	}

	// Em caso de falha do Redis, middleware faz fail-open
	fake.err = errors.New("redis connection refused")
	reqErr := httptest.NewRequest(http.MethodGet, "/ping", nil)
	reqErr.RemoteAddr = "192.168.0.10:12345"
	wErr := httptest.NewRecorder()
	r.ServeHTTP(wErr, reqErr)
	if wErr.Code != http.StatusOK {
		t.Fatalf("esperava fail-open (200) quando Redis falha, retornou %d", wErr.Code)
	}
}

func TestEmailRateLimiterMiddlewares_PreservesBodyAndLimits(t *testing.T) {
	gin.SetMode(gin.TestMode)

	cases := []struct {
		name       string
		middleware func(*EmailRateLimiter) gin.HandlerFunc
	}{
		{
			name:       "EmailRateLimiterMiddleware",
			middleware: EmailRateLimiterMiddleware,
		},
		{
			name:       "EmailRateLimiterLoginMiddleware",
			middleware: EmailRateLimiterLoginMiddleware,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fake := newFakeRedisScripter()
			limiter := NewEmailRateLimiter(fake, "rl:email:test", rate.Every(time.Minute), 1)

			var receivedBody string
			r := gin.New()
			r.POST("/action", tc.middleware(limiter), func(c *gin.Context) {
				b, _ := io.ReadAll(c.Request.Body)
				receivedBody = string(b)
				c.Status(http.StatusOK)
			})

			payload := `{"email":"Test@Example.com"}`

			req1 := httptest.NewRequest(http.MethodPost, "/action", bytes.NewBufferString(payload))
			req1.Header.Set("Content-Type", "application/json")
			w1 := httptest.NewRecorder()
			r.ServeHTTP(w1, req1)
			if w1.Code != http.StatusOK {
				t.Fatalf("primeira requisicao deveria retornar 200, retornou %d", w1.Code)
			}
			if receivedBody != payload {
				t.Fatalf("corpo da requisicao nao foi preservado: got %q want %q", receivedBody, payload)
			}

			req2 := httptest.NewRequest(http.MethodPost, "/action", bytes.NewBufferString(payload))
			req2.Header.Set("Content-Type", "application/json")
			w2 := httptest.NewRecorder()
			r.ServeHTTP(w2, req2)
			if w2.Code != http.StatusTooManyRequests {
				t.Fatalf("segunda requisicao deveria retornar 429, retornou %d", w2.Code)
			}
		})
	}
}
