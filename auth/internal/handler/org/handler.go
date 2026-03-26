package org

import (
	"errors"

	"go.uber.org/zap"

	"github.com/LegationPro/zagforge/auth/internal/db"
	"github.com/LegationPro/zagforge/auth/internal/service/audit"
)

var (
	errInternal          = errors.New("internal error")
	errInvalidUserID     = errors.New("invalid user id")
	errInvalidOrgID      = errors.New("invalid org id")
	errOrgNotFound       = errors.New("organization not found")
	errForbidden         = errors.New("forbidden")
	errNotOwner          = errors.New("only the owner can perform this action")
	errNotAdminOrOwner   = errors.New("admin or owner role required")
	errCannotRemoveOwner = errors.New("cannot remove the org owner")
	errMemberNotFound    = errors.New("member not found")
)

// OrgRole constants.
const (
	RoleOwner  = "owner"
	RoleAdmin  = "admin"
	RoleMember = "member"
)

type Handler struct {
	db       *db.DB
	auditSvc *audit.Service
	log      *zap.Logger
}

func NewHandler(db *db.DB, auditSvc *audit.Service, log *zap.Logger) *Handler {
	return &Handler{db: db, auditSvc: auditSvc, log: log}
}
