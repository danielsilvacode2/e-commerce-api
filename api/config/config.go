package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

/*
DB_USER=root
DB_PASSWORD=minhasenha
DB_NAME=meubanco
DB_HOST=localhost
DB_PORT=3306
API_PORT=:5000
*/

var DbUrl string
var ApiPort string

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
	
	DbUrl = fmt.Sprintf("%s:%s@tcp(%s:%s)/%s?charset=utf8&parseTime=True&loc=Local",
		dbUser,
		dbPassword,
		dbHost,
		dbPort,
		dbName,
	)

}