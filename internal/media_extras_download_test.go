package internal

import (
	"context"
	"testing"
)

func TestShouldDownloadExtraAndFilter(t *testing.T) {
	// extra with missing status and youtube id should be eligible when trailers enabled
	e := Extra{ExtraType: "Trailers", ExtraTitle: "T", YoutubeId: "y1", Status: "missing"}
	cfg := ExtraTypesConfig{Trailers: true}
	if !shouldDownloadExtra(e, cfg) {
		t.Fatalf("expected shouldDownloadExtra true for trailers enabled")
	}

	// disabled in config
	cfg.Trailers = false
	if shouldDownloadExtra(e, cfg) {
		t.Fatalf("expected shouldDownloadExtra false when trailers disabled")
	}
}

func TestFilterAndDownloadEnqueues(t *testing.T) {
	ctx := context.Background()
	// clear queue
	_ = GetRedisClient().Del(ctx, DownloadQueue)

	extras := []Extra{{ExtraType: "Trailers", ExtraTitle: "T", YoutubeId: "qz", Status: "missing"}}
	// call with movie type and ensure enqueue
	filterAndDownloadExtras(MediaTypeMovie, 1, extras, ExtraTypesConfig{Trailers: true})

	// check Redis queue length via LRange (client implementation uses RPush on DownloadQueue)
	// Use internal GetRedisClient to fetch queue content
	client := GetRedisClient()
	res := client.LRange(ctx, DownloadQueue, 0, -1)
	vals, _ := res.Result()
	if len(vals) == 0 {
		t.Fatalf("expected enqueued items in download queue")
	}
}
