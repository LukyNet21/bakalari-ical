package main

import (
	"crypto/rand"
	"fmt"
	"io"
	"log"

	"golang.org/x/crypto/chacha20poly1305"
)

func main() {
	key := make([]byte, chacha20poly1305.KeySize)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		log.Fatalln("key generation error:", err)
	}
	fmt.Println("SAVE THIS KEY to the ENCRYPTION_KEY environment variable for future use.")
	fmt.Printf("%x\n", key)

}
