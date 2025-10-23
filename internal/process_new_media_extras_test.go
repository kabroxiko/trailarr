package internal

import (
	"context"
	"net/http"
	"testing"
	"time"
)

// rewriteTransport rewrites requests to api.themoviedb.org to point to the test server
type rewriteTransport struct {
	base   http.RoundTripper
	target string
}

func (r *rewriteTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// clone request to avoid mutating shared state
	req2 := req.Clone(req.Context())
	// if the request is intended for api.themoviedb.org, rewrite it
	if req2.URL.Host == "api.themoviedb.org" || req2.URL.Host == "api.themoviedb.org:443" {
		req2.URL.Scheme = "http"
		req2.URL.Host = r.target
	}
	return r.base.RoundTrip(req2)
}

func TestProcessNewMediaExtrasEnqueuesTMDBExtras(t *testing.T) {
	ctx := context.Background()

	// clear download queue
	_ = GetRedisClient().Del(ctx, DownloadQueue)

	// Construct a TMDB-like extra directly (deterministic). Use canonical ExtraType to avoid mapping issues.
	extras := []Extra{{ID: "vid1", ExtraType: "Trailers", ExtraTitle: "Official Trailer", YoutubeId: "yt-trailer-1"}}

	// mark downloaded state against an empty mediaPath (no files present)
	mediaPath := t.TempDir()
	MarkDownloadedExtras(extras, mediaPath, "type", "title")

	// Enqueue filtered extras
	cfg := ExtraTypesConfig{Trailers: true}
	filterAndDownloadExtras(MediaTypeMovie, 42, extras, cfg)

	// give a small grace for enqueue work
	time.Sleep(100 * time.Millisecond)

	// verify queue contains the youtube id from the TMDB-like extra
	vals, err := GetRedisClient().LRange(ctx, DownloadQueue, 0, -1).Result()
	if err != nil {
		t.Fatalf("failed to read download queue: %v", err)
	}
	found := false
	for _, v := range vals {
		if v != "" && stringContains(v, "yt-trailer-1") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected download queue to contain queued TMDB youtube id; got: %v", vals)
	}
}

// stringContains is a tiny helper to avoid importing strings in this test file repeatedly
func stringContains(s, sub string) bool {
	return len(s) >= len(sub) && (func() bool { return (indexOf(s, sub) >= 0) })()
}

// indexOf returns the first index of substr in s or -1. Minimal implementation to avoid extra imports.
func indexOf(s, substr string) int {
	if substr == "" {
		return 0
	}
	for i := 0; i+len(substr) <= len(s); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
