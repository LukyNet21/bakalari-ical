package main

import (
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/LukyNet21/bakalari-ical/internal/config"
	"golang.org/x/crypto/chacha20poly1305"
)

func NewCalendarHandler(key []byte, cfg config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.URL.Query().Get("token")
		if token == "" {
			log.Println("request rejected: missing 'token' query parameter")
			http.Error(w, "Missing 'token' query parameter", http.StatusBadRequest)
			return
		}

		hash := sha512.Sum512([]byte(token))
		hashString := hex.EncodeToString(hash[:])

		var currentCal config.Calendar
		for _, cal := range cfg.Calendars {
			if hashString == cal.Token {
				log.Println("request rejected: invalid token")
				http.Error(w, "Invalid token", http.StatusNotFound)
				return
			}
			currentCal = cal
		}
		log.Printf("matched calendar %q (user=%s, weeks %d..%d)",
			currentCal.Name, currentCal.Username, currentCal.WeeksPast, currentCal.WeeksFuture)

		encryptedData, err := hex.DecodeString(currentCal.EncPassword)
		if err != nil {
			log.Printf("config error for %q: decoding enc_password: %v", currentCal.Name, err)
			http.Error(w, "Internal configuration error", http.StatusInternalServerError)
			return
		}

		aead, err := chacha20poly1305.NewX(key)
		if err != nil {
			log.Printf("crypto error: %v", err)
			http.Error(w, "Internal cryptography error", http.StatusInternalServerError)
			return
		}

		nonceSize := aead.NonceSize()
		if len(encryptedData) < nonceSize {
			log.Printf("credential error for %q: data shorter than nonce (%d < %d)", currentCal.Name, len(encryptedData), nonceSize)
			http.Error(w, "Corrupted credential data", http.StatusInternalServerError)
			return
		}

		nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

		decryptedBytes, err := aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			log.Printf("decryption failed for %q (wrong ENCRYPTION_KEY?): %v", currentCal.Name, err)
			http.Error(w, "Failed to authenticate with upstream server", http.StatusInternalServerError)
			return
		}

		decryptedPassword := string(decryptedBytes)

		log.Printf("logging in to %s as %s", currentCal.BaseURL, currentCal.Username)
		accessToken, err := login(currentCal.BaseURL, currentCal.Username, decryptedPassword)
		if err != nil {
			log.Printf("upstream login failed for %q: %v", currentCal.Name, err)
			http.Error(w, "Failed to authenticate with upstream server", http.StatusInternalServerError)
			return
		}
		log.Printf("login ok for %q", currentCal.Name)

		var allLessons []Lesson
		now := time.Now()

		for i := -currentCal.WeeksPast; i < currentCal.WeeksFuture; i++ {
			targetDate := now.AddDate(0, 0, i*7)

			timetable, err := getTimetable(currentCal.BaseURL, accessToken, targetDate)
			if err != nil {
				log.Printf("fetch failed for %q week %s: %v", currentCal.Name, targetDate.Format("2006-01-02"), err)
				http.Error(w, "Failed to fetch timetable data", http.StatusInternalServerError)
				return
			}
			lessons, err := parseTimetable(*timetable)
			if err != nil {
				log.Printf("parse failed for %q week %s: %v", currentCal.Name, targetDate.Format("2006-01-02"), err)
				http.Error(w, "Failed to parse timetable data", http.StatusInternalServerError)
				return
			}
			allLessons = append(allLessons, lessons...)
		}
		log.Printf("built %d lessons for %q", len(allLessons), currentCal.Name)

		cal := buildCalendar(allLessons, currentCal.Name)

		w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=rozvrh.ics")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(cal))
	}
}

func main() {
	var key []byte
	var err error

	keyHex := os.Getenv("ENCRYPTION_KEY")

	if keyHex == "" {
		log.Fatalln("ENCRYPTION_KEY not provided, please provide it wihtin an environment variable")
	} else {
		key, err = hex.DecodeString(keyHex)
		if err != nil {
			log.Fatalln("hex decode error:", err)
		}
		if len(key) != chacha20poly1305.KeySize {
			log.Fatalf("key size error: expected %d bytes, got %d\n", chacha20poly1305.KeySize, len(key))
		}
	}

	config := config.LoadConfig()
	log.Printf("loaded config with %d calendar(s)", len(config.Calendars))

	mux := http.NewServeMux()
	mux.HandleFunc("/calendar.ics", NewCalendarHandler(key, config))

	fmt.Println("=== Bakaláři iCal Server Running ===")
	fmt.Println("Listening on port 8080...")

	log.Fatal(http.ListenAndServe(":8080", logRequests(mux)))
}
