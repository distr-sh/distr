package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

func main() {
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	fmt.Printf("LICENSE_KEY_PRIVATE_KEY=%s\n", base64.StdEncoding.EncodeToString(privKey))
	fmt.Printf("LICENSE_KEY_PUBLIC_KEY=%s\n", base64.StdEncoding.EncodeToString(pubKey))
}
