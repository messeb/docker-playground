// decode-token decrypts a JWE access token issued by Keycloak and pretty-prints
// the inner JWT claims. Useful for inspecting what the API sees after decryption.
//
// Usage:
//
//	go run ./cmd/decode-token <token>
//
// The RSA private key is read from the API_PRIVATE_KEY_BASE64 environment
// variable, or from the ../.env file if the variable is not set.
package main

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/messeb/docker-playground/api-keycloak-security/internal/auth"
)

func main() {
	if len(os.Args) < 2 || os.Args[1] == "" {
		fmt.Fprintln(os.Stderr, "usage: decode-token <token>")
		os.Exit(1)
	}
	tokenStr := os.Args[1]

	keyB64 := os.Getenv("API_PRIVATE_KEY_BASE64")
	if keyB64 == "" {
		keyB64 = readDotEnv("API_PRIVATE_KEY_BASE64")
	}
	if keyB64 == "" {
		fmt.Fprintln(os.Stderr, "error: API_PRIVATE_KEY_BASE64 not found in environment or ../.env")
		os.Exit(1)
	}

	pemBytes, err := base64.StdEncoding.DecodeString(keyB64)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: decode private key: %v\n", err)
		os.Exit(1)
	}

	encKey, err := auth.NewEncryptionKeyFromPEM(pemBytes)
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: load private key: %v\n", err)
		os.Exit(1)
	}

	parts := strings.Split(tokenStr, ".")

	switch len(parts) {
	case 5:
		// JWE compact: header.encryptedKey.iv.ciphertext.tag
		printSeparator()
		fmt.Println("TOKEN TYPE  : JWE (encrypted)")
		printJWEHeader(parts[0])
		printSeparator()

		decrypted, err := encKey.Decrypt([]byte(tokenStr))
		if err != nil {
			fmt.Fprintf(os.Stderr, "error: decryption failed: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("DECRYPTED CLAIMS:")
		printSeparator()
		printJWTClaims(string(decrypted))

	case 3:
		// Plain JWS: header.payload.signature
		printSeparator()
		fmt.Println("TOKEN TYPE  : JWS (signed, not encrypted)")
		printSeparator()
		fmt.Println("CLAIMS:")
		printSeparator()
		printJWTClaims(tokenStr)

	default:
		fmt.Fprintln(os.Stderr, "error: not a valid compact JWT (expected 3 or 5 parts)")
		os.Exit(1)
	}
}

func printJWEHeader(encodedHeader string) {
	headerBytes, err := base64.RawURLEncoding.DecodeString(encodedHeader)
	if err != nil {
		return
	}
	var h map[string]interface{}
	if err := json.Unmarshal(headerBytes, &h); err != nil {
		return
	}
	fmt.Printf("ALGORITHM   : %v (key wrap) + %v (content encryption)\n", h["alg"], h["enc"])
	if kid, ok := h["kid"]; ok {
		fmt.Printf("KEY ID      : %v\n", kid)
	}
}

func printJWTClaims(jws string) {
	parts := strings.Split(jws, ".")
	if len(parts) != 3 {
		fmt.Fprintln(os.Stderr, "error: inner token is not a valid JWS")
		os.Exit(1)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		fmt.Fprintf(os.Stderr, "error: decode payload: %v\n", err)
		os.Exit(1)
	}
	var claims map[string]interface{}
	if err := json.Unmarshal(payload, &claims); err != nil {
		fmt.Fprintf(os.Stderr, "error: parse claims: %v\n", err)
		os.Exit(1)
	}
	out, _ := json.MarshalIndent(claims, "", "  ")
	fmt.Println(string(out))
}

func printSeparator() {
	fmt.Println("─────────────────────────────────────────────")
}

// readDotEnv reads a key=value pair from ../.env (relative to api/ directory).
func readDotEnv(key string) string {
	f, err := os.Open("../.env")
	if err != nil {
		return ""
	}
	defer f.Close()

	prefix := key + "="
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, prefix) {
			return strings.TrimPrefix(line, prefix)
		}
	}
	return ""
}
