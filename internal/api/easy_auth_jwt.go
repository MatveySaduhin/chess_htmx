package api

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

var ErrInvalidEasyAuthToken = errors.New("invalid easy auth token")

type EasyAuthJWTValidator struct {
	jwksURL    string
	issuer     string
	audience   string
	httpClient *http.Client
	cacheTTL   time.Duration

	mu        sync.RWMutex
	keys      map[string]ed25519.PublicKey
	fetchedAt time.Time
}

type EasyAuthClaims struct {
	Subject   string `json:"sub"`
	SessionID string `json:"sid"`
}

type jwtHeader struct {
	Algorithm string `json:"alg"`
	KeyID     string `json:"kid"`
}

type jwtClaims struct {
	Issuer    string          `json:"iss"`
	Subject   string          `json:"sub"`
	Audience  json.RawMessage `json:"aud"`
	ExpiresAt int64           `json:"exp"`
	NotBefore int64           `json:"nbf"`
	SessionID string          `json:"sid"`
}

type jwksResponse struct {
	Keys []jwk `json:"keys"`
}

type jwk struct {
	KeyType string `json:"kty"`
	Curve   string `json:"crv"`
	KeyID   string `json:"kid"`
	Use     string `json:"use"`
	Alg     string `json:"alg"`
	X       string `json:"x"`
}

func NewEasyAuthJWTValidator(config EasyAuthConfig) *EasyAuthJWTValidator {
	return &EasyAuthJWTValidator{
		jwksURL:  config.JWKSURL,
		issuer:   config.Issuer,
		audience: config.Audience,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cacheTTL: 5 * time.Minute,
		keys:     map[string]ed25519.PublicKey{},
	}
}

func (v *EasyAuthJWTValidator) Validate(ctx context.Context, token string) (EasyAuthClaims, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return EasyAuthClaims{}, ErrInvalidEasyAuthToken
	}

	var header jwtHeader
	if err := decodeJWTPart(parts[0], &header); err != nil {
		return EasyAuthClaims{}, fmt.Errorf("%w: invalid header", ErrInvalidEasyAuthToken)
	}
	if header.Algorithm != "EdDSA" || header.KeyID == "" {
		return EasyAuthClaims{}, ErrInvalidEasyAuthToken
	}

	key, err := v.publicKey(ctx, header.KeyID)
	if err != nil {
		return EasyAuthClaims{}, err
	}

	signature, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil {
		return EasyAuthClaims{}, fmt.Errorf("%w: invalid signature encoding", ErrInvalidEasyAuthToken)
	}

	signedPayload := []byte(parts[0] + "." + parts[1])
	if !ed25519.Verify(key, signedPayload, signature) {
		return EasyAuthClaims{}, ErrInvalidEasyAuthToken
	}

	var claims jwtClaims
	if err := decodeJWTPart(parts[1], &claims); err != nil {
		return EasyAuthClaims{}, fmt.Errorf("%w: invalid claims", ErrInvalidEasyAuthToken)
	}

	now := time.Now().Unix()
	if claims.Issuer != v.issuer {
		return EasyAuthClaims{}, ErrInvalidEasyAuthToken
	}
	if !claims.hasAudience(v.audience) {
		return EasyAuthClaims{}, ErrInvalidEasyAuthToken
	}
	if claims.ExpiresAt == 0 || now >= claims.ExpiresAt {
		return EasyAuthClaims{}, ErrInvalidEasyAuthToken
	}
	if claims.NotBefore != 0 && now < claims.NotBefore {
		return EasyAuthClaims{}, ErrInvalidEasyAuthToken
	}
	if claims.Subject == "" || claims.SessionID == "" {
		return EasyAuthClaims{}, ErrInvalidEasyAuthToken
	}

	return EasyAuthClaims{
		Subject:   claims.Subject,
		SessionID: claims.SessionID,
	}, nil
}

func (v *EasyAuthJWTValidator) publicKey(ctx context.Context, kid string) (ed25519.PublicKey, error) {
	if key, ok := v.cachedKey(kid); ok {
		return key, nil
	}

	if err := v.refreshJWKS(ctx); err != nil {
		return nil, err
	}
	if key, ok := v.cachedKey(kid); ok {
		return key, nil
	}

	return nil, ErrInvalidEasyAuthToken
}

func (v *EasyAuthJWTValidator) cachedKey(kid string) (ed25519.PublicKey, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if time.Since(v.fetchedAt) > v.cacheTTL {
		return nil, false
	}
	key, ok := v.keys[kid]
	return key, ok
}

func (v *EasyAuthJWTValidator) refreshJWKS(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return err
	}

	resp, err := v.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch easy auth jwks: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("fetch easy auth jwks: status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return err
	}

	var jwks jwksResponse
	if err := json.Unmarshal(body, &jwks); err != nil {
		return fmt.Errorf("decode easy auth jwks: %w", err)
	}

	keys := make(map[string]ed25519.PublicKey)
	for _, key := range jwks.Keys {
		if key.KeyType != "OKP" || key.Curve != "Ed25519" || key.Alg != "EdDSA" || key.KeyID == "" || key.X == "" {
			continue
		}
		publicKey, err := base64.RawURLEncoding.DecodeString(key.X)
		if err != nil || len(publicKey) != ed25519.PublicKeySize {
			continue
		}
		keys[key.KeyID] = ed25519.PublicKey(publicKey)
	}

	v.mu.Lock()
	defer v.mu.Unlock()
	v.keys = keys
	v.fetchedAt = time.Now()
	return nil
}

func decodeJWTPart(part string, output any) error {
	data, err := base64.RawURLEncoding.DecodeString(part)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}

func (c jwtClaims) hasAudience(expected string) bool {
	var single string
	if err := json.Unmarshal(c.Audience, &single); err == nil {
		return single == expected
	}

	var many []string
	if err := json.Unmarshal(c.Audience, &many); err != nil {
		return false
	}
	for _, audience := range many {
		if audience == expected {
			return true
		}
	}
	return false
}
