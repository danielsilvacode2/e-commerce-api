package auth

import (
	"e-commerce/config"
	"e-commerce/core/apperrors"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type Claims struct {
	Id   uuid.UUID
	Role string
	jwt.RegisteredClaims
}

const tokenTTL = 2 * time.Hour

type revocationStore struct {
	mu           sync.RWMutex
	revokedJTIs  map[string]time.Time
	userRevoked  map[string]time.Time
	cleanupOnce  sync.Once
}

var revocations = &revocationStore{
	revokedJTIs: make(map[string]time.Time),
	userRevoked: make(map[string]time.Time),
}

func (s *revocationStore) startCleanup() {
	s.cleanupOnce.Do(func() {
		go func() {
			ticker := time.NewTicker(15 * time.Minute)
			defer ticker.Stop()
			for now := range ticker.C {
				s.mu.Lock()
				for jti, exp := range s.revokedJTIs {
					if now.After(exp) {
						delete(s.revokedJTIs, jti)
					}
				}
				for uid, ts := range s.userRevoked {
					if now.Sub(ts) > tokenTTL {
						delete(s.userRevoked, uid)
					}
				}
				s.mu.Unlock()
			}
		}()
	})
}

// RevokeToken invalida um JWT especifico (pelo jti) ate o horario de expiracao.
func RevokeToken(jti string, expiresAt time.Time) {
	if jti == "" {
		return
	}
	revocations.startCleanup()
	revocations.mu.Lock()
	defer revocations.mu.Unlock()
	if expiresAt.IsZero() {
		expiresAt = time.Now().Add(tokenTTL)
	}
	revocations.revokedJTIs[jti] = expiresAt
}

// RevokeUserTokensBefore invalida todos os tokens emitidos para um usuario antes do instante atual.
func RevokeUserTokensBefore(userId string) {
	if userId == "" {
		return
	}
	revocations.startCleanup()
	revocations.mu.Lock()
	defer revocations.mu.Unlock()
	revocations.userRevoked[userId] = time.Now()
}

func isTokenRevoked(claims *Claims) bool {
	if claims == nil {
		return true
	}
	revocations.mu.RLock()
	defer revocations.mu.RUnlock()

	if claims.ID != "" {
		if exp, exists := revocations.revokedJTIs[claims.ID]; exists && time.Now().Before(exp) {
			return true
		}
	}

	if revokedAt, exists := revocations.userRevoked[claims.Id.String()]; exists {
		if claims.IssuedAt == nil || !claims.IssuedAt.Time.After(revokedAt) {
			return true
		}
	}

	return false
}

func GenerateToken(id uuid.UUID, role string) (string, error) {
	if config.SecretKey == "" {
		return "", apperrors.ErrUnauthorized
	}

	now := time.Now()
	claims := Claims{
		Id:   id,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.NewString(),
			ExpiresAt: jwt.NewNumericDate(now.Add(tokenTTL)),
			IssuedAt:  jwt.NewNumericDate(now),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	tokenString, err := token.SignedString([]byte(config.SecretKey))

	if err != nil {
		return "", err
	}

	return tokenString, nil
}

func ValidateToken(c *gin.Context) (*Claims, error) {
	if config.SecretKey == "" {
		return nil, apperrors.ErrUnauthorized
	}

	tokenString, err := getToken(c)
	if err != nil || tokenString == "" {
		return nil, apperrors.ErrUnauthorized
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(
		tokenString,
		claims,
		func(token *jwt.Token) (interface{}, error) {
			return []byte(config.SecretKey), nil
		},
		jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}),
	)

	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	if !token.Valid {
		return nil, apperrors.ErrUnauthorized
	}

	if isTokenRevoked(claims) {
		return nil, apperrors.ErrUnauthorized
	}

	return claims, nil
}

func getToken(c *gin.Context) (string, error) {

	tokenString, err := c.Cookie("token")

	if err != nil || tokenString == "" {
		return "", apperrors.ErrUnauthorized
	}

	return tokenString, nil
}
