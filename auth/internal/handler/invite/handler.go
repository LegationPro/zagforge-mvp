package invite

import (
	"errors"

	"go.uber.org/zap"

	"github.com/LegationPro/zagforge/auth/internal/db"
	"github.com/LegationPro/zagforge/auth/internal/service/audit"
)

var (
	errInternal        = errors.New("internal error")
	errInvalidUserID   = errors.New("invalid user id")
	errInvalidOrgID    = errors.New("invalid org id")
	errInvalidInviteID = errors.New("invalid invite id")
	errInvalidToken    = errors.New("invalid or expired invite token")
	errForbidden       = errors.New("admin or owner role required")
	errMaxMembers      = errors.New("organization has reached its member limit")
)

type Handler struct {
	db       *db.DB
	auditSvc *audit.Service
	log      *zap.Logger
}

func NewHandler(db *db.DB, auditSvc *audit.Service, log *zap.Logger) *Handler {
	return &Handler{db: db, auditSvc: auditSvc, log: log}
}
