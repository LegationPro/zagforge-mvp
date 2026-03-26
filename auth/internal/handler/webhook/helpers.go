package webhook

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"slices"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/LegationPro/zagforge/auth/internal/role"
	authstore "github.com/LegationPro/zagforge/auth/internal/store"
	"github.com/LegationPro/zagforge/shared/go/authclaims"
)

func userIDFromContext(r *http.Request) (pgtype.UUID, error) {
	claims, err := authclaims.FromContext(r.Context())
	if err != nil {
		return pgtype.UUID{}, err
	}
	return claims.SubjectUUID()
}

func parseOrgID(r *http.Request) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(chi.URLParam(r, "orgID")); err != nil {
		return id, err
	}
	return id, nil
}

func parseWebhookID(r *http.Request) (pgtype.UUID, error) {
	var id pgtype.UUID
	if err := id.Scan(chi.URLParam(r, "whID")); err != nil {
		return id, err
	}
	return id, nil
}

func (h *Handler) requireOrgAdminOrOwner(r *http.Request, orgID, userID pgtype.UUID) error {
	membership, err := h.db.Queries.GetOrgMembership(r.Context(), authstore.GetOrgMembershipParams{
		OrgID:  orgID,
		UserID: userID,
	})
	if err != nil {
		return errForbidden
	}
	if !slices.Contains(role.OrgAdminOrAbove, membership.Role) {
		return errForbidden
	}
	return nil
}

func generateSecret() (string, error) {
	b := make([]byte, secretBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return secretPrefix + hex.EncodeToString(b), nil
}
