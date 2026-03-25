package zitadelwebhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/netip"

	"github.com/jackc/pgx/v5/pgtype"
	"go.uber.org/zap"

	dbpkg "github.com/LegationPro/zagforge/api/internal/db"
	"github.com/LegationPro/zagforge/shared/go/store"
)

const maxPayloadBytes = 1 << 20 // 1 MB

type Handler struct {
	db     *dbpkg.DB
	secret []byte
	log    *zap.Logger
}

func NewHandler(db *dbpkg.DB, webhookSecret string, log *zap.Logger) *Handler {
	return &Handler{db: db, secret: []byte(webhookSecret), log: log}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	signature := r.Header.Get("X-Zitadel-Signature")

	body, err := io.ReadAll(io.LimitReader(r.Body, maxPayloadBytes+1))
	if err != nil {
		h.log.Error("read body", zap.Error(err))
		http.Error(w, "read error", http.StatusInternalServerError)
		return
	}
	if int64(len(body)) > maxPayloadBytes {
		http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
		return
	}

	// Verify HMAC signature.
	if !h.verifySignature(body, signature) {
		h.log.Warn("invalid webhook signature")
		http.Error(w, "invalid signature", http.StatusUnauthorized)
		return
	}

	// Parse the event envelope.
	var envelope eventEnvelope
	if err := json.Unmarshal(body, &envelope); err != nil {
		h.log.Warn("invalid event payload", zap.Error(err))
		http.Error(w, "invalid payload", http.StatusBadRequest)
		return
	}

	h.log.Info("zitadel webhook", zap.String("event", envelope.Type))

	switch envelope.Type {
	case "user.changed":
		h.handleUserChanged(r, body)
	case "user.removed":
		h.handleUserRemoved(r, body)
	case "session.added":
		h.handleSessionAdded(r, body)
	case "session.removed":
		h.handleSessionRemoved(r, body)
	default:
		h.log.Debug("ignoring event", zap.String("type", envelope.Type))
	}

	w.WriteHeader(http.StatusOK)
}

func (h *Handler) handleUserChanged(r *http.Request, body []byte) {
	var event userEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.log.Warn("parse user event", zap.Error(err))
		return
	}

	if _, err := h.db.Queries.UpsertUser(r.Context(), store.UpsertUserParams{
		ZitadelUserID: event.User.ID,
		Username:      event.User.Username,
		Email:         event.User.Email,
		EmailVerified: event.User.EmailVerified,
		Phone:         pgtype.Text{String: event.User.Phone, Valid: event.User.Phone != ""},
		AvatarUrl:     pgtype.Text{String: event.User.AvatarURL, Valid: event.User.AvatarURL != ""},
	}); err != nil {
		h.log.Error("upsert user from webhook", zap.Error(err), zap.String("zitadel_user_id", event.User.ID))
	}
}

func (h *Handler) handleUserRemoved(r *http.Request, body []byte) {
	var event userEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.log.Warn("parse user removed event", zap.Error(err))
		return
	}

	user, err := h.db.Queries.GetUserByZitadelID(r.Context(), event.User.ID)
	if err != nil {
		h.log.Warn("user not found for removal", zap.String("zitadel_user_id", event.User.ID))
		return
	}

	if err := h.db.Queries.DeleteUser(r.Context(), user.ID); err != nil {
		h.log.Error("delete user from webhook", zap.Error(err))
	}
}

func (h *Handler) handleSessionAdded(r *http.Request, body []byte) {
	var event sessionEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.log.Warn("parse session event", zap.Error(err))
		return
	}

	user, err := h.db.Queries.GetUserByZitadelID(r.Context(), event.Session.UserID)
	if err != nil {
		h.log.Warn("user not found for session", zap.String("zitadel_user_id", event.Session.UserID))
		return
	}

	ip := parseIP(event.Session.IPAddress)

	if _, err := h.db.Queries.UpsertSession(r.Context(), store.UpsertSessionParams{
		UserID:           user.ID,
		ZitadelSessionID: event.Session.ID,
		DeviceName:       pgtype.Text{String: event.Session.UserAgent, Valid: event.Session.UserAgent != ""},
		IpAddress:        ip,
	}); err != nil {
		h.log.Error("upsert session from webhook", zap.Error(err))
	}
}

func (h *Handler) handleSessionRemoved(r *http.Request, body []byte) {
	var event sessionEvent
	if err := json.Unmarshal(body, &event); err != nil {
		h.log.Warn("parse session removed event", zap.Error(err))
		return
	}

	if err := h.db.Queries.DeleteSessionByZitadelID(r.Context(), event.Session.ID); err != nil {
		h.log.Warn("delete session from webhook", zap.Error(err), zap.String("session_id", event.Session.ID))
	}
}

func (h *Handler) verifySignature(body []byte, signature string) bool {
	if signature == "" || len(h.secret) == 0 {
		return false
	}
	mac := hmac.New(sha256.New, h.secret)
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(signature))
}

func parseIP(s string) *netip.Addr {
	if s == "" {
		return nil
	}
	addr, err := netip.ParseAddr(s)
	if err != nil {
		return nil
	}
	return &addr
}

// Event payload types — these match Zitadel's webhook action payloads.
// Field names may vary by Zitadel version; update as needed.

type eventEnvelope struct {
	Type string `json:"type"`
}

type userEvent struct {
	Type string      `json:"type"`
	User userPayload `json:"user"`
}

type userPayload struct {
	ID            string `json:"id"`
	Username      string `json:"userName"`
	Email         string `json:"email"`
	EmailVerified bool   `json:"emailVerified"`
	Phone         string `json:"phone"`
	AvatarURL     string `json:"avatarUrl"`
}

type sessionEvent struct {
	Type    string         `json:"type"`
	Session sessionPayload `json:"session"`
}

type sessionPayload struct {
	ID        string `json:"id"`
	UserID    string `json:"userId"`
	UserAgent string `json:"userAgent"`
	IPAddress string `json:"ipAddress"`
}
