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

func LoadConfig(){
	
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
	
	DbUrl = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

}