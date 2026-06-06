// Package wrapsig HMAC-signs the identity headers the wrapper (authz) emits
// for the upstream API, so the API can prove the request was minted by the
// wrapper and not by a peer process that happened to reach api:8080.
//
// Both authz and api keep a byte-identical copy of this file. The secret is
// shared via the WRAPPER_HMAC_SECRET env var.
package wrapsig

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"sort"
	"strconv"
	"strings"
	"time"
)

// SignedHeaders lists every header that goes into the HMAC. Adding a new
// identity header? Add it here on BOTH sides; the order doesn't matter, the
// canonical-string step sorts.
var SignedHeaders = []string{
	"X-User-Sub",
	"X-User-Email",
	"X-User-Username",
	"X-User-Roles",
	"X-Bank-Account-Number",
	"X-MFA-Verified",
	"X-MFA-Method",
	"X-Wrapper-Timestamp",
}

const (
	HeaderSignature = "X-Wrapper-Signature"
	HeaderTimestamp = "X-Wrapper-Timestamp"
	// MaxSkew bounds replay: a signature is only accepted within ±30s of the
	// API's clock. Demo-grade — production should also bind signatures to the
	// request URI + body hash to remove the window entirely.
	MaxSkew = 30 * time.Second
)

// Sign returns the HMAC-SHA256 over the canonical SignedHeaders string,
// base64url-encoded with no padding.
func Sign(get func(string) string, secret []byte) string {
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(canonical(get)))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// Verify recomputes the HMAC, compares constant-time, and enforces the
// timestamp skew. Returns nil on success.
func Verify(get func(string) string, secret []byte, now time.Time) error {
	provided := get(HeaderSignature)
	if provided == "" {
		return errors.New("missing wrapper signature")
	}
	tsStr := get(HeaderTimestamp)
	if tsStr == "" {
		return errors.New("missing wrapper timestamp")
	}
	ts, err := strconv.ParseInt(strings.TrimSpace(tsStr), 10, 64)
	if err != nil {
		return errors.New("invalid wrapper timestamp")
	}
	delta := now.Unix() - ts
	if delta < 0 {
		delta = -delta
	}
	if time.Duration(delta)*time.Second > MaxSkew {
		return errors.New("wrapper signature outside time window")
	}
	expected := Sign(get, secret)
	if !hmac.Equal([]byte(expected), []byte(provided)) {
		return errors.New("wrapper signature mismatch")
	}
	return nil
}

func canonical(get func(string) string) string {
	lines := make([]string, 0, len(SignedHeaders))
	for _, h := range SignedHeaders {
		// Headers are case-insensitive; lowercase for stability across hops.
		lines = append(lines, strings.ToLower(h)+"="+get(h))
	}
	sort.Strings(lines)
	return strings.Join(lines, "\n")
}
