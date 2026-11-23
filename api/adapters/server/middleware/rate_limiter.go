package middleware

import (
	"strings"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

const (
	rateLimiterTTL     = 1 * time.Hour
	rateLimiterMaxKeys = 50000
)

type limiterEntry struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

type EmailRateLimiter struct {
	emails map[string]*limiterEntry
	mu     *sync.Mutex
	r      rate.Limit
	b      int
}

func NewEmailRateLimiter(r rate.Limit, b int) *EmailRateLimiter {
	rl := &EmailRateLimiter{
		emails: make(map[string]*limiterEntry),
		mu:     &sync.Mutex{},
		r:      r,
		b:      b,
	}
	go rl.cleanupLoop()
	return rl
}

func (e *EmailRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		e.evictStale(time.Now())
	}
}

func (e *EmailRateLimiter) evictStale(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for k, entry := range e.emails {
		if now.Sub(entry.lastSeen) > rateLimiterTTL {
			delete(e.emails, k)
		}
	}
}

func (e *EmailRateLimiter) GetLimiter(email string) *rate.Limiter {
	key := strings.ToLower(strings.TrimSpace(email))
	now := time.Now()

	e.mu.Lock()
	defer e.mu.Unlock()

	if entry, exists := e.emails[key]; exists {
		entry.lastSeen = now
		return entry.limiter
	}

	if len(e.emails) >= rateLimiterMaxKeys {
		for k, entry := range e.emails {
			if now.Sub(entry.lastSeen) > rateLimiterTTL {
				delete(e.emails, k)
			}
		}
		// Se ainda estiver no teto (ex: rajada de chaves novas < TTL), evicta a entrada mais antiga para evitar OOM
		if len(e.emails) >= rateLimiterMaxKeys {
			var oldestKey string
			var oldestTime time.Time
			for k, entry := range e.emails {
				if oldestKey == "" || entry.lastSeen.Before(oldestTime) {
					oldestKey = k
					oldestTime = entry.lastSeen
				}
			}
			if oldestKey != "" {
				delete(e.emails, oldestKey)
			}
		}
	}

	limiter := rate.NewLimiter(e.r, e.b)
	e.emails[key] = &limiterEntry{limiter: limiter, lastSeen: now}
	return limiter
}

type IPRateLimiter struct {
	ips map[string]*limiterEntry
	mu  *sync.Mutex
	r   rate.Limit
	b   int
}

func NewRateLimiter(r rate.Limit, b int) *IPRateLimiter {
	rl := &IPRateLimiter{
		ips: make(map[string]*limiterEntry),
		mu:  &sync.Mutex{},
		r:   r,
		b:   b,
	}
	go rl.cleanupLoop()
	return rl
}

func (i *IPRateLimiter) cleanupLoop() {
	ticker := time.NewTicker(10 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		i.evictStale(time.Now())
	}
}

func (i *IPRateLimiter) evictStale(now time.Time) {
	i.mu.Lock()
	defer i.mu.Unlock()
	for k, entry := range i.ips {
		if now.Sub(entry.lastSeen) > rateLimiterTTL {
			delete(i.ips, k)
		}
	}
}

func (i *IPRateLimiter) GetLimiter(ip string) *rate.Limiter {
	now := time.Now()

	i.mu.Lock()
	defer i.mu.Unlock()

	if entry, exists := i.ips[ip]; exists {
		entry.lastSeen = now
		return entry.limiter
	}

	if len(i.ips) >= rateLimiterMaxKeys {
		for k, entry := range i.ips {
			if now.Sub(entry.lastSeen) > rateLimiterTTL {
				delete(i.ips, k)
			}
		}
		// Se ainda estiver no teto (ex: rajada de IPs novos < TTL), evicta a entrada mais antiga para evitar OOM
		if len(i.ips) >= rateLimiterMaxKeys {
			var oldestKey string
			var oldestTime time.Time
			for k, entry := range i.ips {
				if oldestKey == "" || entry.lastSeen.Before(oldestTime) {
					oldestKey = k
					oldestTime = entry.lastSeen
				}
			}
			if oldestKey != "" {
				delete(i.ips, oldestKey)
			}
		}
	}

	limiter := rate.NewLimiter(i.r, i.b)
	i.ips[ip] = &limiterEntry{limiter: limiter, lastSeen: now}
	return limiter
}
