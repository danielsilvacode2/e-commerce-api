package mysql

import (
	"database/sql"
	"e-commerce/config"
	_ "github.com/go-sql-driver/mysql"
	"log"
)

var DbConnection *sql.DB

func NewMySQLConnection() {

	db, err := sql.Open("mysql", config.DbUrl)
	if err != nil {
		log.Fatal("Error connecting to database: ", err)
	}

	if err = db.Ping(); err != nil {
		log.Fatal("Error pinging database: ", err)
	}

	DbConnection = db
}
