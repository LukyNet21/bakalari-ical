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
			http.Error(w, "Missing 'token' query parameter", http.StatusBadRequest)
			return
		}

		hash := sha512.Sum512([]byte(token))
		hashString := hex.EncodeToString(hash[:])

		var currentCal config.Calendar
		for _, cal := range cfg.Calendars {
			if hashString == cal.Token {
				http.Error(w, "Invalid token", http.StatusNotFound)
				return
			}
			currentCal = cal
		}

		encryptedData, err := hex.DecodeString(currentCal.EncPassword)
		if err != nil {
			http.Error(w, "Internal configuration error", http.StatusInternalServerError)
			return
		}

		aead, err := chacha20poly1305.NewX(key)
		if err != nil {
			http.Error(w, "Internal cryptography error", http.StatusInternalServerError)
			return
		}

		nonceSize := aead.NonceSize()
		if len(encryptedData) < nonceSize {
			http.Error(w, "Corrupted credential data", http.StatusInternalServerError)
			return
		}

		nonce, ciphertext := encryptedData[:nonceSize], encryptedData[nonceSize:]

		decryptedBytes, err := aead.Open(nil, nonce, ciphertext, nil)
		if err != nil {
			http.Error(w, "Failed to authenticate with upstream server", http.StatusInternalServerError)
			return
		}

		decryptedPassword := string(decryptedBytes)

		accessToken, err := login(currentCal.BaseURL, currentCal.Username, decryptedPassword)
		if err != nil {
			http.Error(w, "Failed to authenticate with upstream server", http.StatusInternalServerError)
			return
		}

		var allLessons []Lesson
		now := time.Now()

		for i := -currentCal.WeeksPast; i < currentCal.WeeksFuture; i++ {
			targetDate := now.AddDate(0, 0, i*7)

			timetable, err := getTimetable(currentCal.BaseURL, accessToken, targetDate)
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

	http.HandleFunc("/calendar.ics", NewCalendarHandler(key, config))

	fmt.Println("=== Bakaláři iCal Server Running ===")
	fmt.Println("Listening on port 8080...")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
