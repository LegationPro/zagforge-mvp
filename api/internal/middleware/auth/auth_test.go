package auth_test

import (
	"context"
	"testing"

	"github.com/LegationPro/zagforge/api/internal/middleware/auth"
)

func TestClaimsFromContext_noClaims_returnsError(t *testing.T) {
	_, err := auth.ClaimsFromContext(context.Background())
	if err == nil {
		t.Fatal("expected error when no claims in context")
	}
	if err != auth.ErrClaimsNotFound {
		t.Errorf("expected %q, got %q", auth.ErrClaimsNotFound, err)
	}
}

func TestIsOrgScope_noOrg_returnsFalse(t *testing.T) {
	if auth.IsOrgScope(context.Background()) {
		t.Error("expected IsOrgScope to return false with no org in context")
	}
}

func TestUserIDFromContext_noUser_returnsInvalid(t *testing.T) {
	uid := auth.UserIDFromContext(context.Background())
	if uid.Valid {
		t.Error("expected invalid UUID when no user in context")
	}
}

func TestOrgIDFromContext_noOrg_returnsInvalid(t *testing.T) {
	oid := auth.OrgIDFromContext(context.Background())
	if oid.Valid {
		t.Error("expected invalid UUID when no org in context")
	}
}
