package api

import (
	"os"
	"strings"
)

type EasyAuthConfig struct {
	BaseURL  string
	JWKSURL  string
	Issuer   string
	Audience string
}

func EasyAuthConfigFromEnv() EasyAuthConfig {
	baseURL := strings.TrimRight(os.Getenv("EASY_AUTH_BASE_URL"), "/")
	if baseURL == "" {
		baseURL = "http://localhost:8080"
	}

	jwksURL := os.Getenv("EASY_AUTH_JWKS_URL")
	if jwksURL == "" {
		jwksURL = baseURL + "/.well-known/jwks.json"
	}

	issuer := os.Getenv("EASY_AUTH_ISSUER")
	if issuer == "" {
		issuer = "easy-auth"
	}

	audience := os.Getenv("EASY_AUTH_AUDIENCE")
	if audience == "" {
		audience = "easy-auth-api"
	}

	return EasyAuthConfig{
		BaseURL:  baseURL,
		JWKSURL:  jwksURL,
		Issuer:   issuer,
		Audience: audience,
	}
}
