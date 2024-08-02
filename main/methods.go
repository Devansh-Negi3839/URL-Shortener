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

func getUrlByUrl(w http.ResponseWriter, r *http.Request) {
	db = GetDB()
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
	db = GetDB()
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
