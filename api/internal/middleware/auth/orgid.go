package auth

import (
	"errors"

	"github.com/clerk/clerk-sdk-go/v2"
)

var ErrNoActiveOrg = errors.New("no active organization in session claims")

// ResolveOrgID extracts the active organization ID from session claims.
// Returns ErrNoActiveOrg if the user has no org context in the current session.
// NOTE: This still uses Clerk types until Phase 3 replaces the JWT verification
// with Zitadel OIDC. The function name is updated for consistency.
func ResolveOrgID(claims *clerk.SessionClaims) (string, error) {
	if claims.ActiveOrganizationID == "" {
		return "", ErrNoActiveOrg
	}
	return claims.ActiveOrganizationID, nil
}
