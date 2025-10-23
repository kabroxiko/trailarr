package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSharedExtrasHandlerMergesPersistentAndTMDB(t *testing.T) {
	gin.SetMode(gin.TestMode)
	// prepare persistent extras
	ctx := context.Background()
	// clear extras key in Redis
	_ = GetRedisClient().Del(ctx, ExtrasRedisKey)

	// add a persistent extra
	pe := ExtrasEntry{MediaType: MediaTypeMovie, MediaId: 5, ExtraType: "Trailer", ExtraTitle: "P1", YoutubeId: "y1", Status: "downloaded"}
	if err := AddOrUpdateExtra(ctx, pe); err != nil {
		t.Fatalf("AddOrUpdateExtra failed: %v", err)
	}

	// Do NOT call external TMDB; sharedExtrasHandler will attempt to fetch TMDB extras but may error — persistent extras should still show up.
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	c.Params = gin.Params{gin.Param{Key: "id", Value: "5"}}
	handler := sharedExtrasHandler(MediaTypeMovie)
	handler(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from sharedExtrasHandler, got %d", w.Code)
	}
	var resp map[string][]Extra
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	// response should include the persistent extra
	found := false
	for _, e := range resp["extras"] {
		if e.YoutubeId == "y1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("persistent extra not present in response: %+v", resp)
	}
}

func TestGetMissingExtrasHandlerReturnsMissing(t *testing.T) {
	gin.SetMode(gin.TestMode)
	tmp := t.TempDir()
	cache := filepath.Join(tmp, "wanted.json")
	// item with id 20
	_ = WriteJSONFile(cache, []map[string]interface{}{{"id": 20}})

	// ensure extras collection does not have trailers for id 20
	ctx := context.Background()
	_ = GetRedisClient().Del(ctx, ExtrasRedisKey)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	handler := GetMissingExtrasHandler(cache)
	handler(c)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200 from GetMissingExtrasHandler, got %d", w.Code)
	}
	var resp map[string][]map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if len(resp["items"]) == 0 {
		t.Fatalf("expected missing extras item, got none")
	}
}
