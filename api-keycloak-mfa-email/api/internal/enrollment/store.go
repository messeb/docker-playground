// Package enrollment holds the in-memory state for the in-place MFA
// enrollment flow:
//
//   1. A user without MFA hits a step-up endpoint and gets 403.
//   2. They call /api/v1/mfa/enroll/start with an email — we generate a
//      6-digit code, cache it, and mail it.
//   3. They call /api/v1/mfa/enroll/verify with the code. On success we
//      flip the Keycloak attribute for *future* logins AND record a
//      server-side step-up session so the *current* token can already pass
//      RequireMFA without forcing a re-login.
//
// All state lives in process memory and disappears on restart — fine for the
// demo, not fine for production.
package enrollment

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"time"
)

var (
	ErrNoPending     = errors.New("no enrollment in progress for this user")
	ErrCodeMismatch  = errors.New("OTP code does not match")
	ErrCodeExpired   = errors.New("OTP code expired")
	ErrTooManyTries  = errors.New("too many incorrect attempts — request a new code")
)

const (
	codeTTL     = 5 * time.Minute
	maxAttempts = 5
	// Step-up session validity: how long an in-place enrollment keeps the
	// caller "MFA-verified" for the current access token.
	StepUpTTL = 30 * time.Minute
)

type pending struct {
	code      string
	email     string
	expiresAt time.Time
	tries     int
}

type Store struct {
	mu       sync.Mutex
	pending  map[string]*pending  // key = user sub
	stepUp   map[string]time.Time // sub → MFA-verified until
}

func NewStore() *Store {
	return &Store{
		pending: make(map[string]*pending),
		stepUp:  make(map[string]time.Time),
	}
}

// Start generates a fresh 6-digit code for `sub`, replacing any previous
// pending entry. The code and the email are returned so the caller can mail it.
func (s *Store) Start(sub, email string) (code string, err error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1_000_000))
	if err != nil {
		return "", err
	}
	code = fmt.Sprintf("%06d", n.Int64())

	s.mu.Lock()
	defer s.mu.Unlock()
	s.pending[sub] = &pending{
		code:      code,
		email:     email,
		expiresAt: time.Now().Add(codeTTL),
	}
	return code, nil
}

// Verify checks the supplied code against the cached one. On success it
// clears the pending entry and records a step-up session.
func (s *Store) Verify(sub, code string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[sub]
	if !ok {
		return ErrNoPending
	}
	if time.Now().After(p.expiresAt) {
		delete(s.pending, sub)
		return ErrCodeExpired
	}
	if p.tries >= maxAttempts {
		delete(s.pending, sub)
		return ErrTooManyTries
	}
	if strings.TrimSpace(code) != p.code {
		p.tries++
		return ErrCodeMismatch
	}
	delete(s.pending, sub)
	s.stepUp[sub] = time.Now().Add(StepUpTTL)
	return nil
}

// IsStepUpVerified returns true when the caller is within an active step-up
// window from a successful in-place OTP verification.
func (s *Store) IsStepUpVerified(sub string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	exp, ok := s.stepUp[sub]
	if !ok {
		return false
	}
	if time.Now().After(exp) {
		delete(s.stepUp, sub)
		return false
	}
	return true
}

// PendingEmail returns the email currently registered for an in-flight
// enrollment, for surfacing in the response without exposing the code.
func (s *Store) PendingEmail(sub string) (string, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.pending[sub]
	if !ok {
		return "", false
	}
	return p.email, true
}
