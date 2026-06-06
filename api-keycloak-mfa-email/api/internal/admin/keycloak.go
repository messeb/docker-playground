// Package admin is a thin client for the Keycloak admin REST API.
//
// It is used by the /api/v1/mfa/enable endpoint to flip the user's
// mfa_enabled attribute on their behalf. Demo-only — credentials are the
// master-realm admin user, fetched from env at startup.
package admin

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// KeycloakAdmin is a thread-safe Keycloak admin REST client with a cached
// admin access token.
type KeycloakAdmin struct {
	baseURL     string
	targetRealm string
	user        string
	pass        string
	http        *http.Client

	mu       sync.Mutex
	token    string
	expiry   time.Time
}

func New(baseURL, targetRealm, user, pass string) *KeycloakAdmin {
	return &KeycloakAdmin{
		baseURL:     baseURL,
		targetRealm: targetRealm,
		user:        user,
		pass:        pass,
		http:        &http.Client{Timeout: 10 * time.Second},
	}
}

// EnsureUnmanagedAttributesEnabled flips the realm's user-profile policy so
// arbitrary attributes (mfa_enabled, mfa_method, bank_account_number) can be
// written via the admin REST API. Keycloak 24+ rejects unmanaged attributes by
// default; this is the runtime equivalent of toggling "Unmanaged Attributes"
// in the admin console.
//
// Idempotent — safe to call at every startup.
func (k *KeycloakAdmin) EnsureUnmanagedAttributesEnabled(ctx context.Context) error {
	tok, err := k.adminToken(ctx)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/admin/realms/%s/users/profile", k.baseURL, k.targetRealm)

	getReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	getReq.Header.Set("Authorization", "Bearer "+tok)
	getResp, err := k.http.Do(getReq)
	if err != nil {
		return err
	}
	body, _ := io.ReadAll(getResp.Body)
	getResp.Body.Close()
	if getResp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET users/profile: %d %s", getResp.StatusCode, string(body))
	}
	var profile map[string]any
	if err := json.Unmarshal(body, &profile); err != nil {
		return err
	}
	if profile["unmanagedAttributePolicy"] == "ENABLED" {
		return nil // already set
	}
	profile["unmanagedAttributePolicy"] = "ENABLED"
	updated, _ := json.Marshal(profile)

	putReq, _ := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(updated))
	putReq.Header.Set("Authorization", "Bearer "+tok)
	putReq.Header.Set("Content-Type", "application/json")
	putResp, err := k.http.Do(putReq)
	if err != nil {
		return err
	}
	defer putResp.Body.Close()
	if putResp.StatusCode >= 300 {
		b, _ := io.ReadAll(putResp.Body)
		return fmt.Errorf("PUT users/profile: %d %s", putResp.StatusCode, string(b))
	}
	return nil
}

// EnableEmailMFA sets the user's email and writes the attributes that drive
// the conditional OTP subflow and the mfa_verified / mfa_method claim mappers.
func (k *KeycloakAdmin) EnableEmailMFA(ctx context.Context, userID, email string) error {
	if userID == "" || email == "" {
		return errors.New("userID and email are required")
	}
	tok, err := k.adminToken(ctx)
	if err != nil {
		return fmt.Errorf("admin token: %w", err)
	}

	// PUT /admin/realms/<realm>/users/<id> with the merged user representation.
	// We have to GET first because PUT with a partial body wipes other fields.
	userURL := fmt.Sprintf("%s/admin/realms/%s/users/%s", k.baseURL, k.targetRealm, url.PathEscape(userID))
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, userURL, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := k.http.Do(req)
	if err != nil {
		return fmt.Errorf("GET user: %w", err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("GET user %s: %d %s", userID, resp.StatusCode, string(body))
	}
	var user map[string]any
	if err := json.Unmarshal(body, &user); err != nil {
		return fmt.Errorf("decode user: %w", err)
	}

	user["email"] = email
	user["emailVerified"] = true
	attrs, _ := user["attributes"].(map[string]any)
	if attrs == nil {
		attrs = map[string]any{}
	}
	attrs["mfa_enabled"] = []string{"true"}
	attrs["mfa_method"] = []string{"email"}
	user["attributes"] = attrs

	updated, _ := json.Marshal(user)
	req2, _ := http.NewRequestWithContext(ctx, http.MethodPut, userURL, bytes.NewReader(updated))
	req2.Header.Set("Authorization", "Bearer "+tok)
	req2.Header.Set("Content-Type", "application/json")
	resp2, err := k.http.Do(req2)
	if err != nil {
		return fmt.Errorf("PUT user: %w", err)
	}
	defer resp2.Body.Close()
	if resp2.StatusCode >= 300 {
		b, _ := io.ReadAll(resp2.Body)
		return fmt.Errorf("PUT user %s: %d %s", userID, resp2.StatusCode, string(b))
	}

	// Keycloak's user model cache is what protocol mappers read at token-issue
	// time. Without this invalidation a refresh_token grant immediately after
	// the PUT would still see the *old* attribute set and issue a JWS without
	// mfa_verified.
	if err := k.clearUserCache(ctx, tok); err != nil {
		return fmt.Errorf("clear user cache: %w", err)
	}
	return nil
}

func (k *KeycloakAdmin) clearUserCache(ctx context.Context, tok string) error {
	url := fmt.Sprintf("%s/admin/realms/%s/clear-user-cache", k.baseURL, k.targetRealm)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	resp, err := k.http.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("status %d: %s", resp.StatusCode, string(b))
	}
	return nil
}

func (k *KeycloakAdmin) adminToken(ctx context.Context) (string, error) {
	k.mu.Lock()
	defer k.mu.Unlock()
	if k.token != "" && time.Now().Before(k.expiry) {
		return k.token, nil
	}
	tokenURL := fmt.Sprintf("%s/realms/master/protocol/openid-connect/token", k.baseURL)
	form := url.Values{}
	form.Set("grant_type", "password")
	form.Set("client_id", "admin-cli")
	form.Set("username", k.user)
	form.Set("password", k.pass)
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, tokenURL, bytes.NewBufferString(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := k.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("admin token %d: %s", resp.StatusCode, string(b))
	}
	var tr struct {
		AccessToken string `json:"access_token"`
		ExpiresIn   int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tr); err != nil {
		return "", err
	}
	k.token = tr.AccessToken
	// Refresh 30s before actual expiry to avoid edge races.
	ttl := tr.ExpiresIn - 30
	if ttl < 30 {
		ttl = 30
	}
	k.expiry = time.Now().Add(time.Duration(ttl) * time.Second)
	return k.token, nil
}
