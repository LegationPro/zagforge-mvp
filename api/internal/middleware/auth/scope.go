package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/LegationPro/zagforge/shared/go/authclaims"
	"github.com/LegationPro/zagforge/shared/go/httputil"
)

type orgIDKey struct{}
type userIDKey struct{}

var (
	ErrNoActiveOrg  = errors.New("no active organization in session")
	ErrUserNotFound = errors.New("user not found")
)

// Scope returns middleware that resolves the active workspace from JWT claims.
//
// The org and user IDs come directly from the JWT issued by the auth service.
// org_id is read from the claims.Org.ID field; user_id from claims.Subject.
func Scope(log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := authclaims.FromContext(r.Context())
			if err != nil {
				httputil.ErrResponse(w, http.StatusUnauthorized, ErrClaimsNotFound)
				return
			}

			// Resolve user ID from JWT subject.
			userID, err := claims.SubjectUUID()
			if err != nil {
				log.Warn("scope: invalid user id in claims", zap.String("sub", claims.Subject))
				httputil.ErrResponse(w, http.StatusForbidden, ErrUserNotFound)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, userID)

			// If the JWT includes an org claim, resolve the org scope.
			if claims.Org.ID != "" {
				orgID, err := claims.OrgUUID()
				if err != nil {
					log.Warn("scope: invalid org id in claims", zap.String("org_id", claims.Org.ID))
					httputil.ErrResponse(w, http.StatusForbidden, ErrNoActiveOrg)
					return
				}
				ctx = context.WithValue(ctx, orgIDKey{}, orgID)
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// OrgIDFromContext retrieves the resolved org ID from request context.
// Returns an invalid UUID if the request is scoped to a personal workspace.
func OrgIDFromContext(ctx context.Context) pgtype.UUID {
	id, _ := ctx.Value(orgIDKey{}).(pgtype.UUID)
	return id
}

// UserIDFromContext retrieves the authenticated user's ID from request context.
// Always valid after the Scope middleware has run.
func UserIDFromContext(ctx context.Context) pgtype.UUID {
	id, _ := ctx.Value(userIDKey{}).(pgtype.UUID)
	return id
}

// IsOrgScope returns true if the current request is scoped to an organization.
func IsOrgScope(ctx context.Context) bool {
	return OrgIDFromContext(ctx).Valid
}
