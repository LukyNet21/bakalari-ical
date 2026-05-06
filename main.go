package main

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"time"

	"golang.org/x/crypto/chacha20poly1305"
)

type bakalariCredentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
	BaseURL  string `json:"base_url"`
}

func NewCalendarHandler(key []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		encryptedString := r.URL.Query().Get("credentials")
		if encryptedString == "" {
			http.Error(w, "Missing 'token' query parameter", http.StatusBadRequest)
			return
		}

		encryptedMsg, err := base64.URLEncoding.DecodeString(encryptedString)
		if err != nil {
			http.Error(w, "Invalid token encoding", http.StatusBadRequest)
			return
		}

		aead, err := chacha20poly1305.NewX(key)
		if err != nil {
			http.Error(w, "Internal server error during decryption setup", http.StatusInternalServerError)
			return
		}

		if len(encryptedMsg) < aead.NonceSize() {
			http.Error(w, "Invalid token: payload too short", http.StatusBadRequest)
			return
		}

		nonce := encryptedMsg[:aead.NonceSize()]
		ciphertext := encryptedMsg[aead.NonceSize():]

		plaintext, err := aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			http.Error(w, "Unauthorized: failed to decrypt token", http.StatusUnauthorized)
			return
		}

		var req bakalariCredentials
		if err := json.Unmarshal(plaintext, &req); err != nil {
			http.Error(w, "Invalid token payload structure", http.StatusBadRequest)
			return
		}

		parsedURL, err := url.ParseRequestURI(req.BaseURL)
		if err != nil {
			http.Error(w, "Invalid URL format in token", http.StatusBadRequest)
			return
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			http.Error(w, "URL scheme must be http or https", http.StatusBadRequest)
			return
		}

		safeBaseURL := parsedURL.String()

		accessToken, err := login(safeBaseURL, req.Username, req.Password)
		if err != nil {
			http.Error(w, "Failed to authenticate with upstream server", http.StatusInternalServerError)
			return
		}

		var allLessons []Lesson
		now := time.Now()

		for i := range 6 {
			targetDate := now.AddDate(0, 0, i*7)

			timetable, err := getTimetable(safeBaseURL, accessToken, targetDate)
			if err != nil {
				http.Error(w, "Failed to fetch timetable data", http.StatusInternalServerError)
				return
			}
			lessons, err := parseTimetable(*timetable)
			if err != nil {
				http.Error(w, "Failed to parse timetable data", http.StatusInternalServerError)
				return
			}
			allLessons = append(allLessons, lessons...)
		}

		cal := buildCalendar(allLessons)

		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=rozvrh.ics")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cal))
	}
}

func NewGenerateLinkHandler(key []byte) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method Not Allowed. Please use POST.", http.StatusMethodNotAllowed)
			return
		}

		var req bakalariCredentials
		err := json.NewDecoder(r.Body).Decode(&req)
		if err != nil {
			http.Error(w, "Bad Request: invalid JSON body", http.StatusBadRequest)
			return
		}

		parsedURL, err := url.ParseRequestURI(req.BaseURL)
		if err != nil {
			http.Error(w, "Invalid URL format", http.StatusBadRequest)
			return
		}
		if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
			http.Error(w, "URL scheme must be http or https", http.StatusBadRequest)
			return
		}

		jsonData, err := json.Marshal(req)
		if err != nil {
			http.Error(w, "Internal Server Error: failed to process credentials", http.StatusInternalServerError)
			return
		}

		aead, err := chacha20poly1305.NewX(key)
		if err != nil {
			http.Error(w, "Internal Server Error: encryption setup failed", http.StatusInternalServerError)
			return
		}

		nonce := make([]byte, aead.NonceSize())
		if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
			http.Error(w, "Internal Server Error: failed to generate nonce", http.StatusInternalServerError)
			return
		}

		encryptedMsg := aead.Seal(nonce, nonce, jsonData, nil)
		encryptedString := base64.URLEncoding.EncodeToString(encryptedMsg)

		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(encryptedString))
	}
}

func main() {
	var key []byte
	var err error

	keyHex := os.Getenv("ENCRYPTION_KEY")

	if keyHex == "" {
		key = make([]byte, chacha20poly1305.KeySize)
		if _, err := io.ReadFull(rand.Reader, key); err != nil {
			log.Fatalf("Failed to generate random key: %v", err)
		}

		fmt.Printf("key not provided, using this: %x\nsave it to environment variable for future use\n", key)
	} else {
		key, err = hex.DecodeString(keyHex)
		if err != nil {
			log.Fatalf("Failed to decode hex key: %v", err)
		}
		if len(key) != chacha20poly1305.KeySize {
			log.Fatalf("Invalid key size: expected %d bytes, got %d", chacha20poly1305.KeySize, len(key))
		}
	}

	http.HandleFunc("/generate", NewGenerateLinkHandler(key))
	http.HandleFunc("/calendar.ics", NewCalendarHandler(key))

	fmt.Println("=== Bakaláři iCal Server Running ===")
	fmt.Println("Listening on port 8080...")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
