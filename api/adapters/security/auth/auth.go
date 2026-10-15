package auth

import (
	"e-commerce/config"
	"e-commerce/core/apperrors"
	"strings"
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

func GenerateToken(id uuid.UUID, role string) (string, error) {
	claims := Claims{
		Id:   id,
		Role: role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(2 * time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
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

	tokenString, err := getToken(c)

	if tokenString == "" {
		return nil, apperrors.ErrUnauthorized
	}

	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(config.SecretKey), nil
	})

	if err != nil {
		return nil, apperrors.ErrUnauthorized
	}

	if !token.Valid {
		return nil, apperrors.ErrUnauthorized
	}

	return claims, nil
}

func getToken(c *gin.Context) (string, error) {

	authHeader := c.GetHeader("Authorization")

	if authHeader == "" {
		return "", apperrors.ErrUnauthorized
	}

	parths := strings.Split(authHeader, " ")

	if len(parths) != 2 || parths[0] != "Bearer" {
		return "", apperrors.ErrUnauthorized
	}

	tokenString := parths[1]

	return tokenString, nil
}
