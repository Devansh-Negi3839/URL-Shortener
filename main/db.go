package main

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/sirupsen/logrus"
)

var db *sql.DB

func InitDB() {
	password := os.Getenv("PASSWORD")
	port := os.Getenv("PORT")
	dsn := fmt.Sprintf("root:%s@tcp(localhost:%s)/golang_url", password, port)
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		logrus.Fatal(err)
	}
	fmt.Println("Connected to the database successfully!")
}

func GetDB() *sql.DB {
	return db
}