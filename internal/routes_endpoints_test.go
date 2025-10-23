package internal

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/gin-gonic/gin"
)

// use centralized DoRequest defined in testhelpers_test.go

func TestHealthAndBasicRoutes(t *testing.T) {
	// Ensure clean Redis state for extras and queue
	ctx := context.Background()
	_ = GetRedisClient().Del(ctx, ExtrasRedisKey)
	_ = GetRedisClient().Del(ctx, DownloadQueue)

	r := ginDefaultRouterForTests()

	// Health
	w := DoRequest(r, "GET", "/api/health", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200 for /api/health, got %d", w.Code)
	}
	var h map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &h); err != nil {
		t.Fatalf("failed to decode health body: %v", err)
	}
	if h["status"] != "ok" {
		t.Fatalf("unexpected health status: %v", h)
	}

	// TMDB extratypes
	w = DoRequest(r, "GET", "/api/tmdb/extratypes", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200 for /api/tmdb/extratypes, got %d", w.Code)
	}
	var tmdbResp map[string][]string
	if err := json.Unmarshal(w.Body.Bytes(), &tmdbResp); err != nil {
		t.Fatalf("failed to parse tmdb extratypes: %v", err)
	}
	if _, ok := tmdbResp["tmdbExtraTypes"]; !ok {
		t.Fatalf("tmdb extratypes missing key: %v", tmdbResp)
	}
}

func TestMoviesListAndExtrasEndpoints(t *testing.T) {
	// Use real Redis client if available.
	ctx := context.Background()
	// clear extras and queue
	_ = GetRedisClient().Del(ctx, ExtrasRedisKey)
	_ = GetRedisClient().Del(ctx, DownloadQueue)

	// seed a movie in Redis
	movie := map[string]interface{}{"id": 10, "title": "X", "path": "/tmp/m/10"}
	if err := SaveMediaToRedis(MoviesRedisKey, []map[string]interface{}{movie}); err != nil {
		t.Fatalf("failed to seed movies: %v", err)
	}

	// add an extra in the persistent collection for movie 10
	entry := ExtrasEntry{MediaType: MediaTypeMovie, MediaId: 10, ExtraType: "Trailers", ExtraTitle: "T", YoutubeId: "q1", Status: "missing"}
	if err := AddOrUpdateExtra(context.Background(), entry); err != nil {
		t.Fatalf("failed to add extra: %v", err)
	}

	r := ginDefaultRouterForTests()

	// GET /api/movies
	w := DoRequest(r, "GET", "/api/movies", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200 for /api/movies, got %d", w.Code)
	}
	var listResp map[string][]map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &listResp); err != nil {
		t.Fatalf("failed to parse movies list response: %v", err)
	}
	items := listResp["items"]
	if len(items) == 0 {
		t.Fatalf("expected at least one movie in response")
	}

	// GET specific movie extras
	w = DoRequest(r, "GET", "/api/movies/10/extras", nil)
	if w.Code != 200 {
		t.Fatalf("expected 200 for /api/movies/10/extras, got %d", w.Code)
	}
	var extrasResp map[string][]Extra
	if err := json.Unmarshal(w.Body.Bytes(), &extrasResp); err != nil {
		t.Fatalf("failed to parse extras response: %v", err)
	}
	ex := extrasResp["extras"]
	found := false
	for _, e := range ex {
		if e.YoutubeId == "q1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected persisted extra q1 in extras list; got: %v", ex)
	}
}

// Helper to create a Gin router with RegisterRoutes
func ginDefaultRouterForTests() http.Handler {
	r := gin.New()
	RegisterRoutes(r)
	return r
}
