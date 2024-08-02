package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"math/big"
	"net/http"
	"os"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

const baseShortUrl = "http://shorter.ss/"
const base62Chars = "0123456789ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz"

// Base62Encode encodes a byte slice to a Base62 string.
func base62Encode(bytes []byte) string {
	// Convert byte slice to a big integer
	num := new(big.Int).SetBytes(bytes)

	// Encode the big integer to Base62
	var encoded string
	base := big.NewInt(62)
	zero := big.NewInt(0)
	for num.Cmp(zero) > 0 {
		mod := new(big.Int)
		num.DivMod(num, base, mod)
		encoded = string(base62Chars[mod.Int64()]) + encoded
	}

	return encoded
}

// hashUrl hashes a URL using SHA-256 and returns an 8-character Base62 encoded string.
func hashUrl(url string) string {
	hash := sha256.New()
	hash.Write([]byte(url))
	hashBytes := hash.Sum(nil)

	// Encode to Base62 and truncate to 8 characters
	base62Encoded := base62Encode(hashBytes)
	if len(base62Encoded) > 8 {
		return base62Encoded[:8]
	}
	return base62Encoded
}

type Url struct {
	ShortUrl string `json:"short_url"`
	LongUrl  string `json:"long_url"`
}

var db *sql.DB

func init() {
	if err := godotenv.Load(); err != nil {
		logrus.Print("No .env file found")
	}
}

func main() {
	password := os.Getenv("PASSWORD")
	port := os.Getenv("PORT")
	dsn := fmt.Sprintf("root:%s@tcp(localhost:%s)/golang_url", password, port)
	var err error
	db, err = sql.Open("mysql", dsn)
	if err != nil {
		logrus.Fatal(err)
	}
	defer db.Close()
	fmt.Println("Connected to the database successfully!")

	router := mux.NewRouter()
	router.HandleFunc("/all", getAllUrls).Methods("GET")
	router.HandleFunc("/", getUrlByUrl).Methods("GET")
	router.HandleFunc("/", createUrl).Methods("POST")
	router.HandleFunc("/{shortUrl}", deleteUrl).Methods("DELETE")

	logrus.Fatal(http.ListenAndServe(":8000", router))
}

func getAllUrls(w http.ResponseWriter, r *http.Request) {
	rows, err := db.Query("SELECT short_url, long_url FROM urls")
	if err != nil {
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var results []Url

	for rows.Next() {
		var url Url
		err = rows.Scan(&url.ShortUrl, &url.LongUrl)
		if err != nil {
			logrus.Error(err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			return
		}
		results = append(results, url)
	}

	if err := rows.Err(); err != nil {
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(results); err != nil {
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func getUrlByUrl(w http.ResponseWriter, r *http.Request) {
	var url Url
	if err := json.NewDecoder(r.Body).Decode(&url); err != nil {
		logrus.Error(err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Validate that at least one URL field is provided
	if url.LongUrl == "" && url.ShortUrl == "" {
		http.Error(w, "At least one URL is required to fetch the other", http.StatusBadRequest)
		return
	}

	var query string
	var args []interface{}

	if url.ShortUrl != "" {
		query = "SELECT short_url, long_url FROM urls WHERE short_url = ?"
		args = append(args, url.ShortUrl)
	} else if url.LongUrl != "" {
		query = "SELECT short_url, long_url FROM urls WHERE long_url = ?"
		args = append(args, url.LongUrl)
	}

	row := db.QueryRow(query, args...)

	var result Url
	err := row.Scan(&result.ShortUrl, &result.LongUrl)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Return the result as JSON
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}


func createUrl(w http.ResponseWriter, r *http.Request) {
	var url Url
	if err := json.NewDecoder(r.Body).Decode(&url); err != nil {
		logrus.Error(err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	// Validate LongUrl field
	if url.LongUrl == "" {
		http.Error(w, "Long URL is required", http.StatusBadRequest)
		return
	}

	// Generate a short URL using hashUrl
	shortUrl := hashUrl(url.LongUrl)

	// Insert the new URL into the database
	_, err := db.Exec("INSERT INTO urls (short_url, long_url) VALUES (?, ?)", shortUrl, url.LongUrl)
	if err != nil {
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Prepare the response
	response := map[string]string{
		"short_url": fmt.Sprintf("%s%s", baseShortUrl, shortUrl),
		"long_url":  url.LongUrl,
	}

	// Return the short URL in the response
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}


func deleteUrl(w http.ResponseWriter, r *http.Request) {
	shortUrl := mux.Vars(r)["shortUrl"]

	_, err := db.Exec("DELETE FROM urls WHERE short_url = ?", shortUrl)
	if err != nil {
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
