package account

import (
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	dbpkg "github.com/LegationPro/zagforge/api/internal/db"
	"github.com/LegationPro/zagforge/api/internal/middleware/auth"
	"github.com/LegationPro/zagforge/api/internal/zitadel"
	"github.com/LegationPro/zagforge/shared/go/httputil"
	"github.com/LegationPro/zagforge/shared/go/store"
)

var (
	errInternal        = errors.New("internal error")
	errInvalidBody     = errors.New("invalid request body")
	errSessionNotFound = errors.New("session not found")
)

type Handler struct {
	db      *dbpkg.DB
	zitadel zitadel.Client
	log     *zap.Logger
}

func NewHandler(db *dbpkg.DB, zitadelClient zitadel.Client, log *zap.Logger) *Handler {
	return &Handler{db: db, zitadel: zitadelClient, log: log}
}

// GetProfile returns the authenticated user's profile and org memberships.
func (h *Handler) GetProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	user, err := h.db.Queries.GetUserByID(r.Context(), userID)
	if err != nil {
		h.log.Error("get user", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	memberships, err := h.db.Queries.ListMembershipsByUser(r.Context(), userID)
	if err != nil {
		h.log.Error("list memberships", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	httputil.OkResponse(w, profileResponse{
		User:        user,
		Memberships: memberships,
	})
}

// UpdateProfile updates the authenticated user's username and/or phone.
// Updates Zitadel first (source of truth), then syncs to local DB.
func (h *Handler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	claims, _ := auth.ClaimsFromContext(r.Context())

	body, err := httputil.DecodeJSON[updateProfileRequest](r.Body)
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, errInvalidBody)
		return
	}

	if body.Username == "" && body.Phone == nil {
		httputil.ErrResponse(w, http.StatusBadRequest, errors.New("at least one field required"))
		return
	}

	// Update Zitadel first.
	if err := h.zitadel.UpdateUser(r.Context(), claims.Subject, zitadel.UpdateUserRequest{
		Username: body.Username,
		Phone:    derefStr(body.Phone),
	}); err != nil {
		h.log.Error("zitadel update user", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	// Sync to local DB.
	user, err := h.db.Queries.UpdateUser(r.Context(), store.UpdateUserParams{
		Username:      body.Username,
		Email:         "", // empty = no change (COALESCE in query)
		EmailVerified: false,
		Phone:         pgtype.Text{String: derefStr(body.Phone), Valid: body.Phone != nil},
		AvatarUrl:     pgtype.Text{},
		ID:            userID,
	})
	if err != nil {
		h.log.Error("update user", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	// Audit log.
	if _, err := h.db.Queries.InsertAuditLog(r.Context(), store.InsertAuditLogParams{
		UserID:   userID,
		ActorID:  userID,
		Action:   "account.updated",
		TargetID: userID,
	}); err != nil {
		h.log.Warn("audit log write failed", zap.String("action", "account.updated"), zap.Error(err))
	}

	httputil.OkResponse(w, user)
}

// DeleteAccount deletes the authenticated user from Zitadel and the local DB.
func (h *Handler) DeleteAccount(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())
	claims, _ := auth.ClaimsFromContext(r.Context())

	if err := h.zitadel.DeleteUser(r.Context(), claims.Subject); err != nil {
		h.log.Error("zitadel delete user", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	if err := h.db.Queries.DeleteUser(r.Context(), userID); err != nil {
		h.log.Error("delete user", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// ListSessions returns active sessions for the authenticated user.
func (h *Handler) ListSessions(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	sessions, err := h.db.Queries.ListSessionsByUser(r.Context(), userID)
	if err != nil {
		h.log.Error("list sessions", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	httputil.OkResponse(w, sessions)
}

// RevokeSession terminates a specific session in Zitadel and removes it locally.
func (h *Handler) RevokeSession(w http.ResponseWriter, r *http.Request) {
	userID := auth.UserIDFromContext(r.Context())

	sessionID, err := httputil.ParseUUID(r, "id")
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, err)
		return
	}

	// Delete from local DB (scoped to user).
	if err := h.db.Queries.DeleteSession(r.Context(), store.DeleteSessionParams{
		ID:     sessionID,
		UserID: userID,
	}); err != nil {
		h.log.Error("delete session", zap.Error(err))
		httputil.ErrResponse(w, http.StatusNotFound, errSessionNotFound)
		return
	}

	// Best-effort Zitadel termination — session is already removed locally.
	if err := h.zitadel.TerminateSession(r.Context(), sessionID.String()); err != nil {
		h.log.Warn("zitadel terminate session", zap.Error(err))
	}

	w.WriteHeader(http.StatusNoContent)
}

type profileResponse struct {
	User        store.User                       `json:"user"`
	Memberships []store.ListMembershipsByUserRow `json:"memberships"`
}

type updateProfileRequest struct {
	Username string  `json:"username"`
	Phone    *string `json:"phone"`
}

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
