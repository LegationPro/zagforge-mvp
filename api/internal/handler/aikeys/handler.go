package aikeys

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	dbpkg "github.com/LegationPro/zagforge/api/internal/db"
	"github.com/LegationPro/zagforge/api/internal/middleware/auth"
	"github.com/LegationPro/zagforge/api/internal/service/encryption"
	"github.com/LegationPro/zagforge/shared/go/httputil"
	store "github.com/LegationPro/zagforge/shared/go/store"
)

var (
	errMissingFields = errors.New("provider and raw_key are required")
	errInvalidBody   = errors.New("invalid request body")
	errInternal      = errors.New("internal error")
	errKeyTooShort   = errors.New("raw_key must be at least 8 characters")
)

type Handler struct {
	db  *dbpkg.DB
	enc *encryption.Service
	log *zap.Logger
}

func NewHandler(db *dbpkg.DB, enc *encryption.Service, log *zap.Logger) *Handler {
	return &Handler{db: db, enc: enc, log: log}
}

// List returns provider names and key hints — never the raw key.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	org, err := h.resolveOrg(r)
	if err != nil {
		httputil.ErrResponse(w, http.StatusUnauthorized, err)
		return
	}
	keys, err := h.db.Queries.ListAIProviderKeys(r.Context(), org.ID)
	if err != nil {
		h.log.Error("list ai keys", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}
	httputil.OkResponse(w, keys)
}

// Upsert stores an encrypted AI provider key.
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	body, err := httputil.DecodeJSON[struct {
		Provider string `json:"provider"`
		RawKey   string `json:"raw_key"`
	}](r.Body)
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, errInvalidBody)
		return
	}
	if body.Provider == "" || body.RawKey == "" {
		httputil.ErrResponse(w, http.StatusBadRequest, errMissingFields)
		return
	}
	if len(body.RawKey) < 8 {
		httputil.ErrResponse(w, http.StatusBadRequest, errKeyTooShort)
		return
	}

	cipher, err := h.enc.Encrypt([]byte(body.RawKey))
	if err != nil {
		h.log.Error("encrypt ai key", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	hint := "..." + body.RawKey[len(body.RawKey)-4:]

	org, err := h.resolveOrg(r)
	if err != nil {
		httputil.ErrResponse(w, http.StatusUnauthorized, err)
		return
	}

	if _, err := h.db.Queries.UpsertAIProviderKey(r.Context(), store.UpsertAIProviderKeyParams{
		OrgID:     org.ID,
		Provider:  body.Provider,
		KeyCipher: cipher,
		KeyHint:   hint,
	}); err != nil {
		h.log.Error("upsert ai key", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Delete removes an AI provider key.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	provider := chi.URLParam(r, "provider")
	org, err := h.resolveOrg(r)
	if err != nil {
		httputil.ErrResponse(w, http.StatusUnauthorized, err)
		return
	}
	if err := h.db.Queries.DeleteAIProviderKey(r.Context(), store.DeleteAIProviderKeyParams{
		OrgID: org.ID, Provider: provider,
	}); err != nil {
		h.log.Error("delete ai key", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) resolveOrg(r *http.Request) (store.Organization, error) {
	claims, err := auth.ClaimsFromContext(r.Context())
	if err != nil {
		return store.Organization{}, err
	}
	clerkOrgID, err := auth.ResolveClerkOrgID(claims)
	if err != nil {
		return store.Organization{}, err
	}
	return h.db.Queries.GetOrgByClerkID(r.Context(), clerkOrgID)
}
