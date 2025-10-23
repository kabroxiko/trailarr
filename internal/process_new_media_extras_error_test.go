package internal

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Test that when TMDB returns non-200, processNewMediaExtras does not enqueue downloads
func TestProcessNewMediaExtrasHandlesTMDBError(t *testing.T) {
	// TMDB-like endpoint that returns 500
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	// rewrite transport to point api.themoviedb.org to our test server
	oldTransport := http.DefaultTransport
	http.DefaultTransport = &rewriteTransport{base: oldTransport, target: ts.Listener.Addr().String()}
	defer func() { http.DefaultTransport = oldTransport }()

	// seed config and movies cache
	Config = map[string]interface{}{"general": map[string]interface{}{"tmdbKey": "dummy"}}
	mediaPath := t.TempDir()
	items := []map[string]interface{}{{"id": 7, "tmdbId": 2001, "path": mediaPath}}
	if err := SaveMediaToRedis(MoviesRedisKey, items); err != nil {
		t.Fatalf("failed to seed movies cache: %v", err)
	}

	ctx := context.Background()
	_ = GetRedisClient().Del(ctx, DownloadQueue)

	// run process
	cfg := ExtraTypesConfig{Trailers: true}
	processNewMediaExtras(MediaTypeMovie, 7, cfg)

	// small sleep to allow any enqueues (should not happen)
	time.Sleep(100 * time.Millisecond)

	vals, err := GetRedisClient().LRange(ctx, DownloadQueue, 0, -1).Result()
	if err != nil {
		t.Fatalf("failed to read download queue: %v", err)
	}
	if len(vals) != 0 {
		t.Fatalf("expected no enqueued items when TMDB errors; got: %v", vals)
	}
}
