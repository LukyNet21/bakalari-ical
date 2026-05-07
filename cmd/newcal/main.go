package main

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/LukyNet21/bakalari-ical/internal/config"
	"golang.org/x/crypto/chacha20poly1305"
)

func readInput(prompt string, reader *bufio.Reader) string {
	fmt.Print(prompt)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func readIntInput(prompt string, reader *bufio.Reader) int {
	for {
		input := readInput(prompt, reader)
		val, err := strconv.Atoi(input)
		if err == nil {
			return val
		}
		fmt.Println("  [!] Invalid input. Please enter a valid number.")
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

	reader := bufio.NewReader(os.Stdin)

	fmt.Println("\n====================================")
	fmt.Println("  Calendar Configuration Generator  ")
	fmt.Println("====================================")

	var cal config.Calendar

	cal.Name = readInput("Enter Calendar Name (e.g., Balakáři timetable): ", reader)
	cal.WeeksPast = readIntInput("Enter Weeks Past to sync (e.g., 2): ", reader)
	cal.WeeksFuture = readIntInput("Enter Weeks Future to sync (e.g., 4): ", reader)
	cal.BaseURL = readInput("Enter Base URL (e.g., https://school.bakalari.cz): ", reader)
	cal.Username = readInput("Enter Username: ", reader)

	rawPassword := readInput("Enter Password to Encrypt: ", reader)

	aead, err := chacha20poly1305.NewX(key)
	if err != nil {
		log.Fatalln("crypto setup error:", err)
	}

	nonce := make([]byte, aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		log.Fatalln("nonce generation error:", err)
	}

	ciphertext := aead.Seal(nonce, nonce, []byte(rawPassword), nil)
	cal.EncPassword = hex.EncodeToString(ciphertext)

	tokenBytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, tokenBytes); err != nil {
		log.Fatalln("token generation error:", err)
		os.Exit(1)
	}
	cal.Token = hex.EncodeToString(tokenBytes)

	filename := "config.json"
	var cfg config.Config

	fileData, err := os.ReadFile(filename)
	if err == nil {
		if err := json.Unmarshal(fileData, &cfg); err != nil {
			log.Fatalln("json parse error:", err)
		}
	}

	cfg.Calendars = append(cfg.Calendars, cal)

	jsonData, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		log.Fatalln("json generation error:", err)
	}

	if err = os.WriteFile(filename, jsonData, 0644); err != nil {
		log.Fatalln("file write error:", err)
	}

	fmt.Printf("\nSuccess! Calendar '%s' saved to %s\n", cal.Name, filename)
	fmt.Printf("Generated Auth Token: %s\n", cal.Token)
}
