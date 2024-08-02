package main

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

const baseShortUrl = "http://shorter.ss/"

type Url struct {
	ShortUrl string `json:"short_url"`
	LongUrl  string `json:"long_url"`
}

func init() {
	if err := godotenv.Load(); err != nil {
		logrus.Print("No .env file found")
	}
	InitDB() // Initialize the database connection
}

func main() {
	router := mux.NewRouter()
	router.HandleFunc("/all", getAllUrls).Methods("GET")
	router.HandleFunc("/", getUrlByUrl).Methods("GET")
	router.HandleFunc("/", createUrl).Methods("POST")
	router.HandleFunc("/{shortUrl}", deleteUrl).Methods("DELETE")

	logrus.Fatal(http.ListenAndServe(":8000", router))
}
