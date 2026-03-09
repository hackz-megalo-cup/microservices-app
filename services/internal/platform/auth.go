package platform

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"github.com/go-jose/go-jose/v4"
	"github.com/go-jose/go-jose/v4/jwt"
)

type Claims struct {
	Subject string `json:"sub"`
	Role    string `json:"role"`
	Issuer  string `json:"iss"`
}

type JWTVerifier struct {
	jwksURL   string
	jwks      *jose.JSONWebKeySet
	mu        sync.RWMutex
	lastFetch time.Time
}

func NewJWTVerifier(jwksURL string) *JWTVerifier {
	if jwksURL == "" {
		return nil
	}
	return &JWTVerifier{jwksURL: jwksURL}
}

func (v *JWTVerifier) forceFetchJWKS() error {
	v.mu.Lock()
	v.lastFetch = time.Time{} // invalidate cache
	v.mu.Unlock()
	return v.fetchJWKS()
}

func (v *JWTVerifier) fetchJWKSLocked() error {
	if time.Since(v.lastFetch) < 5*time.Minute {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, v.jwksURL, nil)
	if err != nil {
		return fmt.Errorf("create JWKS request: %w", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Errorf("fetch JWKS: %w", err)
	}
	defer resp.Body.Close()

	var jwks jose.JSONWebKeySet
	if err := json.NewDecoder(resp.Body).Decode(&jwks); err != nil {
		return fmt.Errorf("decode JWKS: %w", err)
	}
	v.jwks = &jwks
	v.lastFetch = time.Now()
	return nil
}

func (v *JWTVerifier) fetchJWKS() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	return v.fetchJWKSLocked()
}

func (v *JWTVerifier) Verify(tokenStr string) (*Claims, error) {
	if v == nil {
		return nil, nil // verification disabled
	}
	if err := v.fetchJWKS(); err != nil {
		slog.Warn("JWKS fetch failed, skipping verification", "error", err)
		return nil, nil
	}

	v.mu.RLock()
	tok, err := jwt.ParseSigned(tokenStr, []jose.SignatureAlgorithm{jose.RS256})
	if err != nil {
		v.mu.RUnlock()
		return nil, fmt.Errorf("parse JWT: %w", err)
	}

	var claims Claims
	var stdClaims jwt.Claims
	if err := tok.Claims(v.jwks, &claims, &stdClaims); err != nil {
		v.mu.RUnlock()
		// kid not found: auth-service may have restarted with new keys; invalidate cache and retry once
		if strings.Contains(err.Error(), "kid not found") || strings.Contains(err.Error(), "matching kid") {
			if retryErr := v.forceFetchJWKS(); retryErr != nil {
				return nil, fmt.Errorf("verify JWT claims: %w", err)
			}
			v.mu.RLock()
			defer v.mu.RUnlock()
			if err := tok.Claims(v.jwks, &claims, &stdClaims); err != nil {
				return nil, fmt.Errorf("verify JWT claims: %w", err)
			}
		} else {
			return nil, fmt.Errorf("verify JWT claims: %w", err)
		}
	} else {
		v.mu.RUnlock()
	}

	if err := stdClaims.Validate(jwt.Expected{
		Issuer: "auth-service",
		Time:   time.Now(),
	}); err != nil {
		return nil, fmt.Errorf("validate JWT: %w", err)
	}

	claims.Subject = stdClaims.Subject
	return &claims, nil
}

func ExtractBearerToken(header string) (string, error) {
	if header == "" {
		return "", errors.New("missing authorization header")
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || parts[0] != "Bearer" {
		return "", errors.New("invalid authorization header format")
	}
	return parts[1], nil
}

type authContextKey struct{}

func ContextWithClaims(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, authContextKey{}, claims)
}

func ClaimsFromContext(ctx context.Context) *Claims {
	c, _ := ctx.Value(authContextKey{}).(*Claims)
	return c
}

// NewAuthInterceptor returns a connect-go interceptor that verifies JWT tokens.
// If verifier is nil, the interceptor is a no-op pass-through.
func NewAuthInterceptor(verifier *JWTVerifier) connect.Interceptor {
	return connect.UnaryInterceptorFunc(func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			if verifier == nil {
				return next(ctx, req)
			}
			token, err := ExtractBearerToken(req.Header().Get("Authorization"))
			if err != nil {
				return next(ctx, req) // no token, let handler decide
			}
			claims, err := verifier.Verify(token)
			if err != nil {
				return nil, connect.NewError(connect.CodeUnauthenticated, err)
			}
			if claims != nil {
				ctx = ContextWithClaims(ctx, claims)
			}
			return next(ctx, req)
		}
	})
}
