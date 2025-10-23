package internal

import (
	"context"
	"encoding/json"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

// TestSettingsPOSTHandlers covers saving radarr/sonarr and general settings via POST
func TestSettingsPOSTHandlers(t *testing.T) {
	tmp := CreateTempConfig(t)
	MediaCoverPath = filepath.Join(tmp, "MediaCover")
	r := NewTestRouter()
	RegisterRoutes(r)

	// POST radarr settings
	payload := `{"url":"http://example.com","apiKey":"kk"}`
	w := DoRequest(r, "POST", radarrSettingsPath, []byte(payload))
	if w.Code != 200 {
		t.Fatalf("expected 200 saving radarr settings, got %d body=%s", w.Code, w.Body.String())
	}

	// POST sonarr settings
	w = DoRequest(r, "POST", "/api/settings/sonarr", []byte(payload))
	if w.Code != 200 {
		t.Fatalf("expected 200 saving sonarr settings, got %d", w.Code)
	}

	// POST general settings (tmdb key and autoDownloadExtras) - handler expects JSON
	genPayload := `{"tmdbKey":"abc","autoDownloadExtras":true}`
	w = DoRequest(r, "POST", "/api/settings/general", []byte(genPayload))
	if w.Code != 200 {
		t.Fatalf("expected 200 saving general settings, got %d body=%s", w.Code, w.Body.String())
	}

	// Read back config file to assert values present
	cfg, err := readConfigFile()
	if err != nil {
		t.Fatalf("failed to read config: %v", err)
	}
	general, ok := cfg["general"].(map[string]interface{})
	if !ok {
		t.Fatalf("missing general section in config: %v", cfg)
	}
	if general["tmdbKey"] != "abc" {
		t.Fatalf("expected tmdbKey saved as abc, got %v", general["tmdbKey"])
	}
	if auto, ok := general["autoDownloadExtras"].(bool); !ok || auto != true {
		t.Fatalf("expected autoDownloadExtras true, got %v", general["autoDownloadExtras"])
	}
}

// TestExtrasDeleteAndExisting exercises delete extras and existing extras listing
func TestExtrasDeleteAndExisting(t *testing.T) {
	tmp := CreateTempConfig(t)

	// seed an extra in Redis persistent store (persistent collection uses ExtrasEntry)
	entry := ExtrasEntry{
		MediaType:  MediaTypeMovie,
		MediaId:    900,
		ExtraType:  "Trailers",
		ExtraTitle: "X",
		YoutubeId:  "y9",
		Status:     "missing",
	}
	if err := AddOrUpdateExtra(context.Background(), entry); err != nil {
		t.Fatalf("failed to seed extra: %v", err)
	}

	r := NewTestRouter()
	RegisterRoutes(r)

	// create a media path and register media in cache so FindMediaPathByID can locate it
	mediaPath := filepath.Join(tmp, "m900")
	_ = os.MkdirAll(filepath.Join(mediaPath, "Trailers"), 0755)
	// create a dummy mkv and meta file so existingExtrasHandler finds it
	_ = os.WriteFile(filepath.Join(mediaPath, "Trailers", "X.mkv"), []byte("x"), 0644)
	meta := `{"extraType":"Trailers","extraTitle":"X","fileName":"X.mkv","youtubeId":"y9","status":"downloaded"}`
	_ = os.WriteFile(filepath.Join(mediaPath, "Trailers", "X.mkv.json"), []byte(meta), 0644)
	movie := map[string]interface{}{"id": 900, "title": "M900", "path": mediaPath}
	if err := SaveMediaToRedis(MoviesRedisKey, []map[string]interface{}{movie}); err != nil {
		t.Fatalf("failed to save media to redis: %v", err)
	}

	// GET existing extras for this moviePath
	w := DoRequest(r, "GET", "/api/extras/existing?moviePath="+url.QueryEscape(mediaPath), nil)
	if w.Code != 200 {
		t.Fatalf("expected 200 listing existing extras, got %d", w.Code)
	}
	var resp map[string][]map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse existing extras response: %v", err)
	}
	if arr, ok := resp["existing"]; !ok || len(arr) == 0 {
		t.Fatalf("expected at least one existing extra, got %v", resp)
	}

	// DELETE the extra (handler expects mediaType and mediaId)
	delPayload := `{"mediaType":"movie","mediaId":900,"youtubeId":"y9"}`
	w = DoRequest(r, "DELETE", "/api/extras", []byte(delPayload))
	if w.Code != 200 {
		t.Fatalf("expected 200 deleting extra, got %d body=%s", w.Code, w.Body.String())
	}

	// Verify it's gone from persistent extras
	remaining, err := GetAllExtras(context.Background())
	if err != nil {
		t.Fatalf("failed to fetch all extras: %v", err)
	}
	for _, e := range remaining {
		if e.YoutubeId == "y9" {
			t.Fatalf("expected extra y9 to be deleted")
		}
	}

	// cleanup
	_ = GetRedisClient().Del(context.Background(), ExtrasRedisKey)
}
