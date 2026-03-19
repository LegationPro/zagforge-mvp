package handler

import (
	"context"
	"net/http"
	"time"

	"github.com/LegationPro/zagforge-mvp-impl/shared/go/httputil"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HealthHandler struct {
	pool *pgxpool.Pool
}

type HealthResponse struct {
	Status string  `json:"status"`
	Reason *string `json:"reason,omitempty"`
}

func NewHealthHandler(pool *pgxpool.Pool) *HealthHandler {
	return &HealthHandler{pool: pool}
}

// Liveness returns 200 if the process is running. No dependency checks.
func (h *HealthHandler) Liveness(w http.ResponseWriter, _ *http.Request) {
	httputil.WriteJSON(w, http.StatusOK, HealthResponse{Status: "ok"})
}

// Readiness returns 200 only if the server can serve traffic (DB reachable).
func (h *HealthHandler) Readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	if err := h.pool.Ping(ctx); err != nil {
		httputil.WriteJSON(w, http.StatusServiceUnavailable, HealthResponse{Status: "unavailable", Reason: new("db unreachable")})
		return
	}

	httputil.WriteJSON(w, http.StatusOK, HealthResponse{Status: "ready"})
}
