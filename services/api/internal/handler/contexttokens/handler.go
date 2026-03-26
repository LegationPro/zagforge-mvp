package contexttokens

import (
	"errors"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	dbpkg "github.com/LegationPro/zagforge/api/internal/db"
	handlerpkg "github.com/LegationPro/zagforge/api/internal/handler"
	"github.com/LegationPro/zagforge/api/internal/middleware/auth"
	"github.com/LegationPro/zagforge/shared/go/httputil"
	store "github.com/LegationPro/zagforge/shared/go/store"
)

var (
	errRepoNotFound  = errors.New("repository not found")
	errInvalidExpiry = errors.New("expires_at must be a valid RFC3339 timestamp")
)

type createTokenResponse struct {
	ID        string     `json:"id"`
	RawToken  string     `json:"raw_token"`
	Label     string     `json:"label,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type Handler struct {
	db  *dbpkg.DB
	log *zap.Logger
}

func NewHandler(db *dbpkg.DB, log *zap.Logger) *Handler {
	return &Handler{db: db, log: log}
}

// List returns context tokens for a repo.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	repoID, err := httputil.ParseUUID(r, "repoID")
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, err)
		return
	}

	orgID := auth.OrgIDFromContext(r.Context())
	if err := handlerpkg.VerifyRepoOwnership(r.Context(), h.db.Queries, repoID, orgID); err != nil {
		httputil.ErrResponse(w, http.StatusNotFound, errRepoNotFound)
		return
	}

	tokens, err := h.db.Queries.ListContextTokensByRepo(r.Context(), repoID)
	if err != nil {
		h.log.Error("list context tokens", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, handlerpkg.ErrInternal)
		return
	}
	httputil.OkResponse(w, tokens)
}

// Create generates a new context token and returns the raw token once only.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFromContext(r.Context())

	repoID, err := httputil.ParseUUID(r, "repoID")
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if err := handlerpkg.VerifyRepoOwnership(r.Context(), h.db.Queries, repoID, orgID); err != nil {
		httputil.ErrResponse(w, http.StatusNotFound, errRepoNotFound)
		return
	}

	body, err := httputil.DecodeJSON[struct {
		Label     string  `json:"label"`
		ExpiresAt *string `json:"expires_at"`
	}](r.Body)
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, handlerpkg.ErrInvalidBody)
		return
	}

	raw, err := generateToken()
	if err != nil {
		h.log.Error("generate token", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, handlerpkg.ErrInternal)
		return
	}
	hash := handlerpkg.SHA256Hash(raw)

	var expiresAt pgtype.Timestamptz
	if body.ExpiresAt != nil {
		t, perr := time.Parse(time.RFC3339, *body.ExpiresAt)
		if perr != nil {
			httputil.ErrResponse(w, http.StatusBadRequest, errInvalidExpiry)
			return
		}
		expiresAt = pgtype.Timestamptz{Time: t, Valid: true}
	}

	tok, err := h.db.Queries.InsertContextToken(r.Context(), store.InsertContextTokenParams{
		RepoID:    repoID,
		OrgID:     orgID,
		TokenHash: hash,
		Label:     pgtype.Text{String: body.Label, Valid: body.Label != ""},
		ExpiresAt: expiresAt,
		// UserID left as zero value (invalid UUID) for org-scoped tokens.
		// Personal workspace tokens will set UserID instead of OrgID.
	})
	if err != nil {
		h.log.Error("insert context token", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, handlerpkg.ErrInternal)
		return
	}

	resp := createTokenResponse{
		ID:        tok.ID.String(),
		RawToken:  raw,
		Label:     tok.Label.String,
		CreatedAt: tok.CreatedAt.Time,
	}
	if tok.ExpiresAt.Valid {
		t := tok.ExpiresAt.Time
		resp.ExpiresAt = &t
	}

	httputil.WriteJSON(w, http.StatusCreated, resp)
}

// Delete revokes a context token (must belong to caller's org).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID := auth.OrgIDFromContext(r.Context())

	tokenID, err := httputil.ParseUUID(r, "tokenID")
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, err)
		return
	}

	if err := h.db.Queries.DeleteContextTokenForOrg(r.Context(), store.DeleteContextTokenForOrgParams{
		ID: tokenID, OrgID: orgID,
	}); err != nil {
		h.log.Error("delete context token", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, handlerpkg.ErrInternal)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
