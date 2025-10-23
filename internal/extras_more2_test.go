package internal

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestScanExtrasInfoAndCanonicalizeMeta(t *testing.T) {
	tmp := t.TempDir()
	sub := filepath.Join(tmp, "Type")
	_ = os.MkdirAll(sub, 0o755)
	metaPath := filepath.Join(sub, "A.mkv.json")
	_ = os.WriteFile(metaPath, []byte(`{"title":"A","fileName":"A.mkv","youtubeId":"y","status":"downloaded"}`), 0o644)
	info := scanExtrasInfo(tmp)
	if len(info) == 0 {
		t.Fatalf("scanExtrasInfo returned empty: %v", info)
	}
}

func TestShouldDownloadExtra(t *testing.T) {
	cfg := ExtraTypesConfig{Trailers: true}
	e := Extra{Status: "missing", YoutubeId: "y1", ExtraType: "Trailers"}
	if !shouldDownloadExtra(e, cfg) {
		t.Fatalf("shouldDownloadExtra expected true for enabled type")
	}
	e.Status = "rejected"
	if shouldDownloadExtra(e, cfg) {
		t.Fatalf("shouldDownloadExtra expected false for rejected status")
	}
	e.Status = "missing"
	e.YoutubeId = ""
	if shouldDownloadExtra(e, cfg) {
		t.Fatalf("shouldDownloadExtra expected false for missing youtube id")
	}
}

func TestHandleExtraDownloadEnqueues(t *testing.T) {
	ctx := context.Background()
	// purge queue
	_ = GetRedisClient().Del(ctx, DownloadQueue)
	e := Extra{Status: "missing", YoutubeId: "q1", ExtraType: "Trailers", ExtraTitle: "T"}
	// call handleExtraDownload - should enqueue via AddToDownloadQueue
	if err := handleExtraDownload(MediaTypeMovie, 1, e); err != nil {
		t.Fatalf("handleExtraDownload returned error: %v", err)
	}
	// ensure queue has item
	q := GetCurrentDownloadQueue()
	if len(q) == 0 {
		t.Fatalf("expected queue to have item after handleExtraDownload")
	}
}
