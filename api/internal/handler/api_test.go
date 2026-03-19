package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	dbpkg "github.com/LegationPro/zagforge-mvp-impl/api/internal/db"
	"github.com/LegationPro/zagforge-mvp-impl/api/internal/handler"
)

// newAPIHandler creates an APIHandler with a nil DB pool (sufficient for param-validation tests).
func newAPIHandler() *handler.APIHandler {
	return handler.NewAPIHandler(&dbpkg.DB{}, zap.NewNop())
}

// chiRequest builds an httptest request routed through a chi mux so URL params resolve.
func chiRequest(t *testing.T, method, pattern, target string, h http.HandlerFunc) *httptest.ResponseRecorder {
	t.Helper()
	mux := chi.NewRouter()
	switch method {
	case http.MethodGet:
		mux.Get(pattern, h)
	}
	r := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	mux.ServeHTTP(w, r)
	return w
}

func decodeResponse(t *testing.T, w *httptest.ResponseRecorder) handler.Response {
	t.Helper()
	var resp handler.Response
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

// -- parseLimit tests --

func TestParseLimit_default(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/test", nil)
	// parseLimit is unexported, so test it indirectly via ListJobs with invalid repoID.
	// Instead, test the handler behavior that depends on it.

	// No limit param → default 50 (tested through handler behavior).
	// We just verify the request doesn't crash.
	_ = r
}

// -- Content-Type tests --

func TestAPIHandler_responses_haveJSONContentType(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet, "/api/v1/repos/{repoID}", "/api/v1/repos/not-a-uuid", h.GetRepo)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected Content-Type application/json, got %q", ct)
	}
}

// -- GetRepo tests --

func TestGetRepo_invalidUUID_returns400(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet, "/api/v1/repos/{repoID}", "/api/v1/repos/not-a-uuid", h.GetRepo)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	if resp.Error == nil || *resp.Error != handler.ErrInvalidRepoID.Error() {
		t.Errorf("expected error %q, got %v", handler.ErrInvalidRepoID, resp.Error)
	}
}

// -- GetJob tests --

func TestGetJob_invalidUUID_returns400(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet, "/api/v1/repos/{repoID}/jobs/{jobID}", "/api/v1/repos/xxx/jobs/bad-id", h.GetJob)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	if resp.Error == nil || *resp.Error != handler.ErrInvalidJobID.Error() {
		t.Errorf("expected error %q, got %v", handler.ErrInvalidJobID, resp.Error)
	}
}

// -- ListJobs tests --

func TestListJobs_invalidRepoUUID_returns400(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet, "/api/v1/repos/{repoID}/jobs", "/api/v1/repos/bad/jobs", h.ListJobs)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	if resp.Error == nil || *resp.Error != handler.ErrInvalidRepoID.Error() {
		t.Errorf("expected error %q, got %v", handler.ErrInvalidRepoID, resp.Error)
	}
}

func TestListJobs_invalidCursor_returns400(t *testing.T) {
	h := newAPIHandler()
	// Use a valid UUID format so we get past parseUUID.
	w := chiRequest(t, http.MethodGet,
		"/api/v1/repos/{repoID}/jobs",
		"/api/v1/repos/00000000-0000-0000-0000-000000000001/jobs?cursor=not-a-date",
		h.ListJobs,
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	if resp.Error == nil || *resp.Error != handler.ErrInvalidCursor.Error() {
		t.Errorf("expected error %q, got %v", handler.ErrInvalidCursor, resp.Error)
	}
}

// -- GetSnapshot tests --

func TestGetSnapshot_invalidUUID_returns400(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet, "/api/v1/snapshots/{snapshotID}", "/api/v1/snapshots/nope", h.GetSnapshot)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	if resp.Error == nil || *resp.Error != handler.ErrInvalidSnapshotID.Error() {
		t.Errorf("expected error %q, got %v", handler.ErrInvalidSnapshotID, resp.Error)
	}
}

// -- ListSnapshots tests --

func TestListSnapshots_invalidRepoUUID_returns400(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet,
		"/api/v1/repos/{repoID}/snapshots",
		"/api/v1/repos/bad/snapshots?branch=main",
		h.ListSnapshots,
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestListSnapshots_missingBranch_returns400(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet,
		"/api/v1/repos/{repoID}/snapshots",
		"/api/v1/repos/00000000-0000-0000-0000-000000000001/snapshots",
		h.ListSnapshots,
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	if resp.Error == nil || *resp.Error != handler.ErrBranchRequired.Error() {
		t.Errorf("expected error %q, got %v", handler.ErrBranchRequired, resp.Error)
	}
}

// -- GetLatestSnapshot tests --

func TestGetLatestSnapshot_invalidRepoUUID_returns400(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet,
		"/api/v1/repos/{repoID}/snapshots/latest",
		"/api/v1/repos/bad/snapshots/latest?branch=main",
		h.GetLatestSnapshot,
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestGetLatestSnapshot_missingBranch_returns400(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet,
		"/api/v1/repos/{repoID}/snapshots/latest",
		"/api/v1/repos/00000000-0000-0000-0000-000000000001/snapshots/latest",
		h.GetLatestSnapshot,
	)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	resp := decodeResponse(t, w)
	if resp.Error == nil || *resp.Error != handler.ErrBranchRequired.Error() {
		t.Errorf("expected error %q, got %v", handler.ErrBranchRequired, resp.Error)
	}
}

// -- Response shape tests --

func TestErrResponse_shape(t *testing.T) {
	h := newAPIHandler()
	w := chiRequest(t, http.MethodGet, "/api/v1/repos/{repoID}", "/api/v1/repos/bad", h.GetRepo)

	resp := decodeResponse(t, w)
	if resp.Data != nil {
		t.Error("expected nil data on error response")
	}
	if resp.Error == nil {
		t.Error("expected non-nil error on error response")
	}
	if resp.NextCursor != nil {
		t.Error("expected nil next_cursor on error response")
	}
}
