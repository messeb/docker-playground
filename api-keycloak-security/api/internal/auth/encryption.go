package auth

import (
	"crypto"
	_ "crypto/sha256" // register SHA-256 for JWK thumbprint
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	"github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// EncryptionKey holds the RSA private key used to decrypt JWE access tokens.
// Keycloak encrypts tokens with the matching public key, fetched from
// GET /.well-known/jwks.json on this API. Only this API can decrypt them.
type EncryptionKey struct {
	private *rsa.PrivateKey
	public  jwk.Key
}

// NewEncryptionKeyFromPEM parses a PEM-encoded RSA private key (PKCS#1 or PKCS#8)
// and derives the JWK representation of the public key used for the JWKS endpoint.
func NewEncryptionKeyFromPEM(pemBytes []byte) (*EncryptionKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil {
		return nil, fmt.Errorf("no PEM block found in private key data")
	}

	var rsaKey *rsa.PrivateKey
	switch block.Type {
	case "RSA PRIVATE KEY":
		pk, err := x509.ParsePKCS1PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS1 private key: %w", err)
		}
		rsaKey = pk
	case "PRIVATE KEY":
		pk, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("parse PKCS8 private key: %w", err)
		}
		var ok bool
		rsaKey, ok = pk.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("PKCS8 key is not RSA")
		}
	default:
		return nil, fmt.Errorf("unsupported PEM block type: %q", block.Type)
	}

	pubJWK, err := jwk.FromRaw(rsaKey.Public())
	if err != nil {
		return nil, fmt.Errorf("create JWK from public key: %w", err)
	}
	pubJWK.Set(jwk.AlgorithmKey, jwa.RSA_OAEP)
	pubJWK.Set(jwk.KeyUsageKey, "enc")

	// Use the SHA-256 thumbprint as the key ID so Keycloak can match keys by kid.
	thumb, err := pubJWK.Thumbprint(crypto.SHA256)
	if err == nil {
		pubJWK.Set(jwk.KeyIDKey, base64.RawURLEncoding.EncodeToString(thumb))
	}

	return &EncryptionKey{private: rsaKey, public: pubJWK}, nil
}

// Decrypt decrypts a JWE compact token and returns the plaintext (the inner JWS).
func (e *EncryptionKey) Decrypt(token []byte) ([]byte, error) {
	plain, err := jwe.Decrypt(token, jwe.WithKey(jwa.RSA_OAEP, e.private))
	if err != nil {
		return nil, fmt.Errorf("JWE decryption failed: %w", err)
	}
	return plain, nil
}

// PublicJWK returns the public RSA key as a JWK, for the /.well-known/jwks.json endpoint.
func (e *EncryptionKey) PublicJWK() jwk.Key {
	return e.public
}
