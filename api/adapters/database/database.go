package database

import (
	"database/sql"
	"e-commerce/config"
	"log"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

var DB_MySQL *sql.DB

func NewMySQLConnection() {

	db, err := sql.Open("mysql", config.DbUrl)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Error pinging database: ", err)
	}


	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(25)
	db.SetConnMaxIdleTime(time.Minute * 4)
	db.SetConnMaxLifetime(time.Minute * 7)

	DB_MySQL = db
}
