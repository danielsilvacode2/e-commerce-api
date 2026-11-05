package middleware

import (
	"strings"
	"sync"

	"golang.org/x/time/rate"
)

type EmailRateLimiter struct {
	emails map[string]*rate.Limiter
	mu     *sync.Mutex
	r      rate.Limit
	b      int
}

func NewEmailRateLimiter(r rate.Limit, b int) *EmailRateLimiter {
	return &EmailRateLimiter{
		emails: make(map[string]*rate.Limiter),
		mu:     &sync.Mutex{},
		r:      r,
		b:      b,
	}
}

func (e *EmailRateLimiter) GetLimiter(email string) *rate.Limiter {
	key := strings.ToLower(strings.TrimSpace(email))

	e.mu.Lock()
	defer e.mu.Unlock()

	limiter, exists := e.emails[key]
	if !exists {
		limiter = rate.NewLimiter(e.r, e.b)
		e.emails[key] = limiter
	}

	return limiter
}

type IPRateLimiter struct {
	ips map[string]*rate.Limiter
	mu  *sync.Mutex
	r   rate.Limit
	b   int
}

func NewRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	return &IPRateLimiter{
		ips: make(map[string]*rate.Limiter),
		mu:  &sync.Mutex{},
		r:   r,
		b:   b,
	}
}

func (i *IPRateLimiter) AddIP(ip string) *rate.Limiter {
	i.mu.Lock()
	defer i.mu.Unlock()

	limiter, exists := i.ips[ip]

	if !exists {
		limiter = rate.NewLimiter(i.r, i.b)
		i.ips[ip] = limiter
	}

	return limiter
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	i.mu.Lock()

	limiter, exists := i.ips[ip]

	if !exists {
		i.mu.Unlock()
		return i.AddIP(ip)
	}

	i.mu.Unlock()
	return limiter
}
