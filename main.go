package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"
)

func calendarHandler(w http.ResponseWriter, r *http.Request) {
	baseURL := r.URL.Query().Get("base_url")
	username := r.URL.Query().Get("username")
	password := r.URL.Query().Get("password")

	if baseURL == "" {
		http.Error(w, "Missing 'base_url' parameter", http.StatusBadRequest)
		return
	}
	if username == "" {
		http.Error(w, "Missing 'username' parameter", http.StatusBadRequest)
		return
	}
	if password == "" {
		http.Error(w, "Missing 'password' parameter", http.StatusBadRequest)
		return
	}

	parsedURL, err := url.ParseRequestURI(baseURL)
	if err != nil {
		http.Error(w, "Invalid URL format", http.StatusBadRequest)
		return
	}
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		http.Error(w, "URL scheme must be http or https", http.StatusBadRequest)
		return
	}

	safeBaseURL := parsedURL.String()

	accessToken, err := login(safeBaseURL, username, password)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}

	var allLessons []Lesson
	now := time.Now()

	for i := range 6 {
		targetDate := now.AddDate(0, 0, i*7)

		timetable, err := getTimetable(baseURL, accessToken, targetDate)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		lessons, err := parseTimetable(*timetable)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
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

func main() {
	http.HandleFunc("/calendar.ics", calendarHandler)

	fmt.Println("=== Bakaláři iCal Server běží ===")
	fmt.Println("Naslouchám na portu 8080...")

	log.Fatal(http.ListenAndServe(":8080", nil))
}
