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

var RedisURL string

var EmailHost string
var EmailPort string
var EmailPassword string
var EmaiUser string
var EmailFrom string
var BrevoAPIKey string

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

	DbUrl = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8mb4&parseTime=True&loc=Local&tls=true",
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
	EmailFrom = strings.TrimSpace(os.Getenv("EMAIL_FROM"))
	BrevoAPIKey = strings.TrimSpace(os.Getenv("BREVO_API_KEY"))

	rabbitMQUser := os.Getenv("RABBITMQ_USER")
	rabbitMQPassword := os.Getenv("RABBITMQ_PASSWORD")
	rabbitMQHost := os.Getenv("RABBITMQ_HOST")
	rabbitMQPort := os.Getenv("RABBITMQ_PORT")
	rabbitMQVhost := os.Getenv("RABBITMQ_VHOST")

	protocol := "amqps"
	if rabbitMQPort == "5672" {
		protocol = "amqp"
	}

	RabbitMQUrl = fmt.Sprintf("%s://%s:%s@%s:%s/%s",
		protocol,
		rabbitMQUser,
		rabbitMQPassword,
		rabbitMQHost,
		rabbitMQPort,
		rabbitMQVhost,
	)

	RedisURL = os.Getenv("REDIS_URL")

	

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
		AllowedOrigins = []string{
			"http://localhost:3000",
			"https://nero-pi-teal.vercel.app",
		}
	} else {
		for _, o := range strings.Split(originsEnv, ",") {
			if trimmed := strings.TrimRight(strings.TrimSpace(o), "/"); trimmed != "" {
				AllowedOrigins = append(AllowedOrigins, trimmed)
			}
		}
	}
}
