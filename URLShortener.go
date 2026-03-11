package main

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"time"
)

type URL struct {
	ID           string    `json:"id"`
	OriginalURL  string    `json:"original_url"`
	ShortURL     string    `json:"short_url"`
	CreationDate time.Time `json:"creation_date"`
}

// In-memory URL storage
var urlDB = make(map[string]URL)

// Base folder where HTML and CSS are located
var baseFolder = "URL_SHORTENER"

// Normalize URL (add https:// if missing)
func normalizeURL(originalURL string) string {
	if len(originalURL) < 4 || originalURL[:4] != "http" {
		return "https://" + originalURL
	}
	return originalURL
}

// Generate short code
func GenerateShortURL(originalURL string) string {
	originalURL = normalizeURL(originalURL)
	hash := md5.Sum([]byte(originalURL))
	return hex.EncodeToString(hash[:])[:8]
}

// Save URL in memory
func saveURL(originalURL string) string {
	originalURL = normalizeURL(originalURL)
	// check if already exists
	for _, u := range urlDB {
		if u.OriginalURL == originalURL {
			return u.ID
		}
	}

	shortID := GenerateShortURL(originalURL)
	urlDB[shortID] = URL{
		ID:           shortID,
		OriginalURL:  originalURL,
		ShortURL:     shortID,
		CreationDate: time.Now(),
	}
	return shortID
}

// Retrieve original URL
func getOriginalURL(id string) (string, error) {
	u, ok := urlDB[id]
	if !ok {
		return "", errors.New("URL not found")
	}
	return u.OriginalURL, nil
}

// Serve Index.html
func RootPageURL(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	indexPath := filepath.Join(baseFolder, "Index.html")
	http.ServeFile(w, r, indexPath)
}

// Handle POST /shorten
func ShortURLHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var data struct {
		URL string `json:"url"`
	}
	err := json.NewDecoder(r.Body).Decode(&data)
	if err != nil || data.URL == "" {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	shortID := saveURL(data.URL)
	resp := struct {
		ShortURL string `json:"short_url"`
	}{
		ShortURL: "/redirect/" + shortID,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// Redirect /redirect/{id} to original URL
func RedirectHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/redirect/"):]
	if id == "" {
		http.Error(w, "Invalid URL", http.StatusBadRequest)
		return
	}

	originalURL, err := getOriginalURL(id)
	if err != nil {
		http.Error(w, "URL not found", http.StatusNotFound)
		return
	}

	http.Redirect(w, r, originalURL, http.StatusFound)
}

func main() {
	// Serve CSS and any other static files from URL_SHORTENER folder
	fs := http.FileServer(http.Dir(baseFolder))
	http.Handle("/style.css", fs)

	// Routes
	http.HandleFunc("/", RootPageURL)
	http.HandleFunc("/shorten", ShortURLHandler)
	http.HandleFunc("/redirect/", RedirectHandler)

	fmt.Println("Server running at http://localhost:3000")
	err := http.ListenAndServe(":3000", nil)
	if err != nil {
		fmt.Println("Error starting server:", err)
	}
}