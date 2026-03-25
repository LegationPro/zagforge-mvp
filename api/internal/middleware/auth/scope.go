package auth

import (
	"context"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	"github.com/LegationPro/zagforge/shared/go/httputil"
	"github.com/LegationPro/zagforge/shared/go/store"
)

type orgIDKey struct{}
type userIDKey struct{}

var (
	ErrNoActiveOrg  = errors.New("no active organization in session")
	ErrUserNotFound = errors.New("user not found")
)

// Scope returns middleware that resolves the active workspace from JWT claims.
//
// If the JWT contains an org claim (urn:zitadel:iam:org:id), the request is
// scoped to that organization and org_id is stored in context.
//
// If no org claim is present, the request is scoped to the user's personal
// workspace and user_id is stored in context.
//
// Both user_id and org_id are always available via their respective context
// helpers — handlers use whichever is valid for the current scope.
func Scope(queries *store.Queries, log *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims, err := ClaimsFromContext(r.Context())
			if err != nil {
				httputil.ErrResponse(w, http.StatusUnauthorized, ErrClaimsNotFound)
				return
			}

			// Always resolve the user from the JWT subject.
			user, err := queries.GetUserByZitadelID(r.Context(), claims.Subject)
			if err != nil {
				if errors.Is(err, pgx.ErrNoRows) {
					log.Warn("scope: user not found", zap.String("zitadel_user_id", claims.Subject))
					httputil.ErrResponse(w, http.StatusForbidden, ErrUserNotFound)
					return
				}
				log.Error("scope: get user", zap.Error(err))
				httputil.ErrResponse(w, http.StatusInternalServerError, errors.New("internal error"))
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey{}, user.ID)

			// If the JWT includes an org claim, resolve the org scope.
			if claims.OrgID != "" {
				org, err := queries.GetOrgByZitadelID(r.Context(), claims.OrgID)
				if err != nil {
					log.Warn("scope: org not found", zap.String("zitadel_org_id", claims.OrgID))
					httputil.ErrResponse(w, http.StatusForbidden, ErrNoActiveOrg)
					return
				}
				ctx = context.WithValue(ctx, orgIDKey{}, org.ID)
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
