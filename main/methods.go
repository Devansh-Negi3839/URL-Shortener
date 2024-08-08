package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"

	_ "github.com/go-sql-driver/mysql"
	"github.com/gorilla/mux"
	"github.com/sirupsen/logrus"
)

func getAllUrls(w http.ResponseWriter, r *http.Request) {
	db = GetDB()
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

func getLongUrlByShortUrl(w http.ResponseWriter, r *http.Request) {
	db = GetDB()

	shortUrl := mux.Vars(r)["shorturl"]

	query := "SELECT short_url, long_url FROM urls WHERE short_url = ?"

	row := db.QueryRow(query, shortUrl)

	var result Url
	err := row.Scan(&result.ShortUrl, &result.LongUrl)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Not Found", http.StatusNotFound)
			return
		}
		logrus.Println(err) // Use log.Println instead of logrus if you don't have logrus setup
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, result.LongUrl, http.StatusMovedPermanently)
}

func createUrl(w http.ResponseWriter, r *http.Request) {
	db := GetDB()
	var url Url
	if err := json.NewDecoder(r.Body).Decode(&url); err != nil {
		logrus.Error("Failed to decode JSON:", err)
		http.Error(w, "Bad Request", http.StatusBadRequest)
		return
	}

	if url.LongUrl == "" {
		http.Error(w, "Long URL is required", http.StatusBadRequest)
		return
	}

	// Check if the LongUrl already exists in the database
	var existingUrl Url
	query := "SELECT short_url, long_url FROM urls WHERE long_url = ?"
	row := db.QueryRow(query, url.LongUrl)
	err := row.Scan(&existingUrl.ShortUrl, &existingUrl.LongUrl)

	if err == nil {
		// If the URL exists, return the existing short URL
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		if err := json.NewEncoder(w).Encode(map[string]string{
			"short_url": fmt.Sprintf("%s%s", baseShortUrl, existingUrl.ShortUrl),
			"long_url":  existingUrl.LongUrl,
		}); err != nil {
			logrus.Error("Failed to encode JSON:", err)
			http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		}
		return
	}

	if err != sql.ErrNoRows {
		// Handle SQL errors other than "no rows"
		logrus.Error("Database query failed:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Generate a new short URL
	shortUrl := hashUrl(url.LongUrl)

	// Insert the new URL into the database
	insertQuery := "INSERT INTO urls (short_url, long_url) VALUES (?, ?)"
	_, err = db.Exec(insertQuery, shortUrl, url.LongUrl)
	if err != nil {
		logrus.Error("Failed to insert new URL:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Respond with the newly created short URL
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	if err := json.NewEncoder(w).Encode(map[string]string{
		"short_url": fmt.Sprintf("%s%s", baseShortUrl, shortUrl),
		"long_url":  url.LongUrl,
	}); err != nil {
		logrus.Error("Failed to encode JSON:", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func deleteUrl(w http.ResponseWriter, r *http.Request) {
	db = GetDB()
	shortUrl := mux.Vars(r)["shortUrl"]

	_, err := db.Exec("DELETE FROM urls WHERE short_url = ?", shortUrl)
	if err != nil {
		logrus.Error(err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
