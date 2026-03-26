package webhook

import (
	"net/http"

	"go.uber.org/zap"

	authstore "github.com/LegationPro/zagforge/auth/internal/store"
	"github.com/LegationPro/zagforge/auth/internal/validate"
	"github.com/LegationPro/zagforge/shared/go/httputil"
)

type createResponse struct {
	Webhook webhookResponse `json:"webhook"`
	Secret  string          `json:"secret"`
}

// Create registers a new webhook subscription.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	actorID, err := userIDFromContext(r)
	if err != nil {
		httputil.ErrResponse(w, http.StatusUnauthorized, errInvalidUserID)
		return
	}

	orgID, err := parseOrgID(r)
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, errInvalidOrgID)
		return
	}

	if err := h.requireOrgAdminOrOwner(r, orgID, actorID); err != nil {
		httputil.ErrResponse(w, http.StatusForbidden, err)
		return
	}

	body, err := httputil.DecodeJSON[createWebhookRequest](r.Body)
	if err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, err)
		return
	}
	if err := validate.Struct(body); err != nil {
		httputil.ErrResponse(w, http.StatusBadRequest, err)
		return
	}

	secret, err := generateSecret()
	if err != nil {
		h.log.Error("generate webhook secret", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	sub, err := h.db.Queries.CreateWebhookSubscription(r.Context(), authstore.CreateWebhookSubscriptionParams{
		OrgID:  orgID,
		Url:    body.URL,
		Secret: secret,
		Events: body.Events,
	})
	if err != nil {
		h.log.Error("create webhook", zap.Error(err))
		httputil.ErrResponse(w, http.StatusInternalServerError, errInternal)
		return
	}

	httputil.WriteJSON(w, http.StatusCreated, httputil.Response[createResponse]{
		Data: createResponse{
			Webhook: toWebhookResponse(sub),
			Secret:  secret,
		},
	})
}
