package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

var DbUrl string
var ApiPort string
var SecretKey string

var EmailHost string
var EmailPort string
var EmailPassword string
var EmaiUser string
var EmailFrom string

func LoadConfig() {

	if err := godotenv.Load(); err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbHost := os.Getenv("DB_HOST")
	dbPort := os.Getenv("DB_PORT")
	ApiPort = os.Getenv("API_PORT")
	SecretKey = os.Getenv("SECRET_KEY")
	EmaiUser = os.Getenv("EMAIL_USER")
	EmailHost = os.Getenv("EMAIL_HOST")
	EmailPassword = os.Getenv("EMAIL_PASSWORD")
	EmailPort = os.Getenv("EMAIL_PORT")
	EmailFrom = os.Getenv("EMAIL_FROM")

	DbUrl = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

}
