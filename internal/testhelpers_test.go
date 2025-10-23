package internal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

// CreateTempConfig creates a temp config directory and returns its path and sets TrailarrRoot/ConfigPath.
func CreateTempConfig(t *testing.T) string {
	tmp := t.TempDir()
	oldRoot := TrailarrRoot
	oldConfig := ConfigPath
	t.Cleanup(func() {
		TrailarrRoot = oldRoot
		ConfigPath = oldConfig
	})
	TrailarrRoot = tmp
	cfgDir := filepath.Join(TrailarrRoot, "config")
	if err := os.MkdirAll(cfgDir, 0755); err != nil {
		t.Fatalf("failed to create config dir: %v", err)
	}
	ConfigPath = filepath.Join(cfgDir, "config.yml")
	// Write a minimal config file so code that expects sections won't panic.
	// Use the same defaults as production helpers.
	minimal := map[string]interface{}{
		"general":    DefaultGeneralConfig(),
		"ytdlpFlags": DefaultYtdlpFlagsConfig(),
	}
	if err := writeConfigFile(minimal); err != nil {
		t.Fatalf("failed to write initial config file: %v", err)
	}
	return tmp
}

// WriteConfig writes content to the current ConfigPath.
func WriteConfig(t *testing.T, content []byte) {
	if err := os.WriteFile(ConfigPath, content, 0644); err != nil {
		t.Fatalf("failed to write config file: %v", err)
	}
}

// NewTestRouter returns a new Gin engine in release mode for tests.
func NewTestRouter() *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	return gin.New()
}

// DoRequest is a small helper to make HTTP requests against a handler.
func DoRequest(r http.Handler, method, path string, body []byte) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, bytes.NewReader(body))
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}
