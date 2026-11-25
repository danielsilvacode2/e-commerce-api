package config

import (
	"errors"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
)

var DbUrl string
var ApiPort string
var SecretKey string

var RabbitMQUrl string

var RedisUser string
var RedisHost string
var RedisPort string
var RedisPassword string

var EmailHost string
var EmailPort string
var EmailPassword string
var EmaiUser string
var EmailFrom string

var AccessTokenPayment string
var WebhookSecret string
var AllowPaymentSimulation bool

var KeyOpenRouter string
var OpenRouterModel string

var CookieSecure bool
var CookieDomain string
var AllowedOrigins []string

const minSecretKeyLength = 32

func LoadConfig() {

	if err := godotenv.Load(); err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("[ERROR] falha ao ler arquivo .env : %v", err)
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")

	DbUrl = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

	ApiPort = strings.TrimSpace(os.Getenv("API_PORT"))
	if ApiPort == "" {
		ApiPort = ":5000"
	} else if !strings.Contains(ApiPort, ":") {
		ApiPort = ":" + ApiPort
	}
	SecretKey = strings.TrimSpace(os.Getenv("SECRET_KEY"))
	if len(SecretKey) < minSecretKeyLength {
		log.Fatalf("[ERROR] SECRET_KEY deve estar configurada e possuir no minimo %d caracteres", minSecretKeyLength)
	}

	EmaiUser = os.Getenv("EMAIL_USER")
	EmailHost = os.Getenv("EMAIL_HOST")
	EmailPassword = os.Getenv("EMAIL_PASSWORD")
	EmailPort = os.Getenv("EMAIL_PORT")
	EmailFrom = os.Getenv("EMAIL_FROM")

	rabbitMQUser := os.Getenv("RABBITMQ_USER")
	rabbitMQPassword := os.Getenv("RABBITMQ_PASSWORD")
	rabbitMQHost := os.Getenv("RABBITMQ_HOST")
	rabbitMQPort := os.Getenv("RABBITMQ_PORT")

	RabbitMQUrl = fmt.Sprintf("amqp://%s:%s@%s:%s/",
		rabbitMQUser,
		rabbitMQPassword,
		rabbitMQHost,
		rabbitMQPort,
	)

	RedisUser = strings.TrimSpace(os.Getenv("REDIS_USER"))
	RedisHost = strings.TrimSpace(os.Getenv("REDIS_HOST"))
	if RedisHost == "" {
		RedisHost = "localhost"
	}
	RedisPort = strings.TrimSpace(os.Getenv("REDIS_PORT"))
	if RedisPort == "" {
		RedisPort = "6379"
	}
	RedisPassword = os.Getenv("REDIS_PASSWORD")

	AccessTokenPayment = strings.TrimSpace(os.Getenv("ACCESS_TOKEN_PAYMENT"))
	WebhookSecret = strings.TrimSpace(os.Getenv("MERCADO_PAGO_WEBHOOK_SECRET"))
	AllowPaymentSimulation = strings.EqualFold(strings.TrimSpace(os.Getenv("ALLOW_PAYMENT_SIMULATION")), "true")

	KeyOpenRouter = strings.TrimSpace(os.Getenv("KEY_OPEN_ROUTER"))
	OpenRouterModel = strings.TrimSpace(os.Getenv("OPEN_ROUTER_MODEL"))
	if OpenRouterModel == "" {
		OpenRouterModel = "google/gemini-2.5-flash-lite"
	}

	CookieSecure = strings.EqualFold(strings.TrimSpace(os.Getenv("COOKIE_SECURE")), "true")
	CookieDomain = strings.TrimSpace(os.Getenv("COOKIE_DOMAIN"))

	originsEnv := strings.TrimSpace(os.Getenv("CORS_ALLOWED_ORIGINS"))
	if originsEnv == "" {
		AllowedOrigins = []string{"http://localhost:3000"}
	} else {
		for _, o := range strings.Split(originsEnv, ",") {
			if trimmed := strings.TrimSpace(o); trimmed != "" {
				AllowedOrigins = append(AllowedOrigins, trimmed)
			}
		}
	}
}
