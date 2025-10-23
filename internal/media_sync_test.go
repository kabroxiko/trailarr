package internal

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestSyncMediaCacheWritesCache(t *testing.T) {
	// provider server returns two items
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		items := []map[string]interface{}{{"id": 11, "title": "A"}, {"id": 12, "title": "B"}}
		_ = json.NewEncoder(w).Encode(items)
	}))
	defer srv.Close()

	tmp := t.TempDir()
	cfg := map[string]interface{}{"radarr": map[string]interface{}{"url": srv.URL, "apiKey": ""}}
	cfgPath := filepath.Join(tmp, "cfg.yml")
	data, _ := json.Marshal(cfg)
	_ = os.WriteFile(cfgPath, data, 0644)
	oldCfg := ConfigPath
	ConfigPath = cfgPath
	defer func() { ConfigPath = oldCfg }()

	cacheFile := filepath.Join(tmp, "cache.json")
	if err := SyncMediaCache("radarr", "/api/v3/movie", cacheFile, func(m map[string]interface{}) bool { return true }); err != nil {
		t.Fatalf("SyncMediaCache failed: %v", err)
	}
	if _, err := os.Stat(cacheFile); err != nil {
		t.Fatalf("expected cache file to exist: %v", err)
	}
	var items []map[string]interface{}
	if err := ReadJSONFile(cacheFile, &items); err != nil {
		t.Fatalf("failed to read cache file: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}
}
