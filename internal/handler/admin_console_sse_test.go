package handler

import (
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"

	"github.com/manimovassagh/rampart/internal/middleware"
)

func TestDashboardSSERejectsWhenGlobalLimitReached(t *testing.T) {
	h := &AdminConsoleHandler{}
	// Simulate global limit already reached.
	h.sseGlobal.Store(maxSSEGlobal)

	authUser := &middleware.AuthenticatedUser{UserID: uuid.New(), OrgID: uuid.New(), Roles: []string{"admin"}}
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/sse", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.DashboardSSE(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	// Counter should not have increased.
	if got := h.sseGlobal.Load(); got != maxSSEGlobal {
		t.Errorf("global counter = %d, want %d", got, maxSSEGlobal)
	}
}

func TestDashboardSSERejectsWhenPerUserLimitReached(t *testing.T) {
	h := &AdminConsoleHandler{}
	userID := uuid.New()

	// Simulate per-user limit already reached.
	counter := &atomic.Int32{}
	counter.Store(maxSSEPerUser)
	h.ssePerUser.Store(userID, counter)

	authUser := &middleware.AuthenticatedUser{UserID: userID, OrgID: uuid.New(), Roles: []string{"admin"}}
	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/sse", http.NoBody)
	ctx := middleware.SetAuthenticatedUser(req.Context(), authUser)
	req = req.WithContext(ctx)
	w := httptest.NewRecorder()

	h.DashboardSSE(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	// Global counter should have been decremented back.
	if got := h.sseGlobal.Load(); got != 0 {
		t.Errorf("global counter = %d, want 0 (should be decremented)", got)
	}
}

func TestDashboardSSERejectsUnauthenticated(t *testing.T) {
	h := &AdminConsoleHandler{}

	req := httptest.NewRequest(http.MethodGet, "/admin/dashboard/sse", http.NoBody)
	w := httptest.NewRecorder()

	h.DashboardSSE(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}
