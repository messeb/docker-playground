package jwe

import (
	"crypto"
	_ "crypto/sha256" // register SHA-256 for JWK thumbprint
	"crypto/rsa"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"

	"github.com/lestrrat-go/jwx/v2/jwa"
	jwelib "github.com/lestrrat-go/jwx/v2/jwe"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

// BFFKey is the RSA private key that *decrypts* the JWE Keycloak emits.
// Keycloak fetches the matching public key from GET /.well-known/jwks.json on
// this authorizer and uses it to wrap the access token.
type BFFKey struct {
	private *rsa.PrivateKey
	public  jwk.Key
}

// NewBFFKey parses a PEM-encoded RSA private key (PKCS#1 or PKCS#8) and
// derives the public JWK used for the authz JWKS endpoint.
func NewBFFKey(pemBytes []byte) (*BFFKey, error) {
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
	if thumb, err := pubJWK.Thumbprint(crypto.SHA256); err == nil {
		pubJWK.Set(jwk.KeyIDKey, base64.RawURLEncoding.EncodeToString(thumb))
	}
	return &BFFKey{private: rsaKey, public: pubJWK}, nil
}

// Decrypt unwraps a JWE compact token and returns the inner JWS plaintext.
func (b *BFFKey) Decrypt(token []byte) ([]byte, error) {
	plain, err := jwelib.Decrypt(token, jwelib.WithKey(jwa.RSA_OAEP, b.private))
	if err != nil {
		return nil, fmt.Errorf("JWE decryption failed: %w", err)
	}
	return plain, nil
}

// Encrypt wraps a plaintext (typically a JWS compact string) as a JWE using
// this key's *public* half. Symmetric counterpart to Decrypt — used by the
// /wrap-token endpoint to hand the SPA a JWE-form bearer.
func (b *BFFKey) Encrypt(plaintext []byte) (string, error) {
	jwe, err := jwelib.Encrypt(
		plaintext,
		jwelib.WithKey(jwa.RSA_OAEP, &b.private.PublicKey),
		jwelib.WithContentEncryption(jwa.A256GCM),
	)
	if err != nil {
		return "", fmt.Errorf("JWE encryption failed: %w", err)
	}
	return string(jwe), nil
}

// PublicJWKSetJSON returns the serialized JWKS document for the JWKS endpoint.
func (b *BFFKey) PublicJWKSetJSON() ([]byte, error) {
	ks := jwk.NewSet()
	ks.AddKey(b.public)
	return json.Marshal(ks)
}
