package org

import (
	"context"
	"slices"

	"github.com/jackc/pgx/v5/pgtype"

	authstore "github.com/LegationPro/zagforge/auth/internal/store"
)

// requireRole checks that the user has one of the allowed roles in the org.
func (h *Handler) requireRole(ctx context.Context, orgID, userID pgtype.UUID, allowed ...string) error {
	membership, err := h.db.Queries.GetOrgMembership(ctx, authstore.GetOrgMembershipParams{
		OrgID:  orgID,
		UserID: userID,
	})
	if err != nil {
		return errForbidden
	}

	if !slices.Contains(allowed, membership.Role) {
		if slices.Contains(allowed, RoleOwner) && len(allowed) == 1 {
			return errNotOwner
		}
		return errNotAdminOrOwner
	}

	return nil
}
