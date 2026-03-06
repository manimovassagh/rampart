package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manimovassagh/rampart/internal/plugin"
)

type stubPluginLister struct {
	plugins []plugin.PluginInfo
}

func (s *stubPluginLister) Plugins() []plugin.PluginInfo {
	return s.plugins
}

func TestListPluginsReturnsPlugins(t *testing.T) {
	lister := &stubPluginLister{
		plugins: []plugin.PluginInfo{
			{
				Name:        "test-plugin",
				Version:     "1.0.0",
				Description: "A test plugin",
				Hooks:       []plugin.HookPoint{plugin.HookPreAuth},
			},
		},
	}

	h := NewPluginHandler(lister)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/plugins", nil)
	rec := httptest.NewRecorder()

	h.ListPlugins(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("expected content-type application/json, got %s", ct)
	}

	var result []plugin.PluginInfo
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 1 {
		t.Fatalf("expected 1 plugin, got %d", len(result))
	}
	if result[0].Name != "test-plugin" {
		t.Errorf("expected name test-plugin, got %s", result[0].Name)
	}
	if result[0].Version != "1.0.0" {
		t.Errorf("expected version 1.0.0, got %s", result[0].Version)
	}
}

func TestListPluginsReturnsEmptyArray(t *testing.T) {
	lister := &stubPluginLister{plugins: nil}
	h := NewPluginHandler(lister)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/plugins", nil)
	rec := httptest.NewRecorder()

	h.ListPlugins(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var result []plugin.PluginInfo
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("expected 0 plugins, got %d", len(result))
	}
}
