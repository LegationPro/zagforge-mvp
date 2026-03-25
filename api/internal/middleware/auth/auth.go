package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/MicahParks/keyfunc/v3"
	"github.com/golang-jwt/jwt/v5"
	"go.uber.org/zap"

	"github.com/LegationPro/zagforge/shared/go/httputil"
)

type contextKey string

const claimsKey contextKey = "zitadel_claims"

var (
	ErrMissingToken   = errors.New("missing authorization token")
	ErrInvalidToken   = errors.New("invalid or expired token")
	ErrClaimsNotFound = errors.New("session claims not found in context")
)

// Claims represents the JWT claims from a Zitadel-issued OIDC token.
type Claims struct {
	jwt.RegisteredClaims
	// Zitadel includes the active org ID in this custom claim when the user
	// has selected an organization context.
	OrgID string `json:"urn:zitadel:iam:org:id,omitempty"`
}

// ClaimsFromContext retrieves the session claims from the request context.
func ClaimsFromContext(ctx context.Context) (*Claims, error) {
	claims, ok := ctx.Value(claimsKey).(*Claims)
	if !ok {
		return nil, ErrClaimsNotFound
	}
	return claims, nil
}

// JWKSConfig holds the configuration needed to verify Zitadel JWTs.
type JWKSConfig struct {
	IssuerURL string // e.g. "https://auth.zagforge.com"
	ProjectID string // expected audience
}

// Auth returns middleware that verifies Zitadel OIDC JWTs on incoming requests.
// It fetches the JWKS from the issuer's well-known endpoint and validates tokens locally.
func Auth(cfg JWKSConfig, log *zap.Logger) (func(http.Handler) http.Handler, error) {
	jwksURL := strings.TrimRight(cfg.IssuerURL, "/") + "/oauth/v2/keys"

	jwks, err := keyfunc.NewDefault([]string{jwksURL})
	if err != nil {
		return nil, err
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			token := extractToken(r)
			if token == "" {
				httputil.ErrResponse(w, http.StatusUnauthorized, ErrMissingToken)
				return
			}

			claims := &Claims{}
			parsed, err := jwt.ParseWithClaims(token, claims, jwks.KeyfuncCtx(r.Context()),
				jwt.WithIssuer(cfg.IssuerURL),
				jwt.WithAudience(cfg.ProjectID),
				jwt.WithExpirationRequired(),
			)
			if err != nil || !parsed.Valid {
				log.Warn("auth: invalid token", zap.Error(err))
				httputil.ErrResponse(w, http.StatusUnauthorized, ErrInvalidToken)
				return
			}

			ctx := context.WithValue(r.Context(), claimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}, nil
}

func extractToken(r *http.Request) string {
	token, found := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
	if !found {
		return ""
	}
	return token
}
