package webhook

import (
	"errors"

	"go.uber.org/zap"

	"github.com/LegationPro/zagforge/auth/internal/db"
)

const (
	defaultDeliveryLimit = int32(50)
	secretPrefix         = "whsec_"
	secretBytes          = 32
)

var (
	errInternal      = errors.New("internal error")
	errInvalidUserID = errors.New("invalid user id")
	errInvalidOrgID  = errors.New("invalid org id")
	errInvalidID     = errors.New("invalid id")
	errForbidden     = errors.New("admin or owner role required")
)

type Handler struct {
	db  *db.DB
	log *zap.Logger
}

func NewHandler(db *db.DB, log *zap.Logger) *Handler {
	return &Handler{db: db, log: log}
}
