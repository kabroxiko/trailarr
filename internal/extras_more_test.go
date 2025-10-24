package internal

import (
	"context"
	"testing"
)

func TestAddGetRemoveExtraLifecycle(t *testing.T) {
	ctx := context.Background()
	// ensure clean keys - best-effort
	_ = GetStoreClient().Del(ctx, ExtrasStoreKey)
	// Add
	e := ExtrasEntry{MediaType: MediaTypeMovie, MediaId: 100, YoutubeId: "tadd", ExtraTitle: "T", ExtraType: "Trailers", Status: "downloaded"}
	if err := AddOrUpdateExtra(ctx, e); err != nil {
		t.Fatalf("AddOrUpdateExtra failed: %v", err)
	}
	got, err := GetExtraByYoutubeId(ctx, "tadd", MediaTypeMovie, 100)
	if err != nil || got == nil {
		t.Fatalf("GetExtraByYoutubeId failed to retrieve: %v %v", err, got)
	}
	if got.YoutubeId != "tadd" {
		t.Fatalf("unexpected youtube id: %v", got)
	}
	// Remove
	if err := RemoveExtra(ctx, "tadd", MediaTypeMovie, 100); err != nil {
		t.Fatalf("RemoveExtra failed: %v", err)
	}
	got2, err := GetExtraByYoutubeId(ctx, "tadd", MediaTypeMovie, 100)
	if err != nil {
		t.Fatalf("GetExtraByYoutubeId after remove returned error: %v", err)
	}
	if got2 != nil {
		t.Fatalf("expected nil after remove, got: %v", got2)
	}
}

func TestRemoveAll429RejectionsRemovesEntry(t *testing.T) {
	ctx := context.Background()
	_ = GetStoreClient().Del(ctx, ExtrasStoreKey)
	e := ExtrasEntry{MediaType: MediaTypeMovie, MediaId: 200, YoutubeId: "t429", ExtraTitle: "X", ExtraType: "Scenes", Status: "rejected", Reason: "HTTP 429 Too Many"}
	if err := AddOrUpdateExtra(ctx, e); err != nil {
		t.Fatalf("AddOrUpdateExtra failed: %v", err)
	}
	// Sanity: ensure it exists
	if got, _ := GetExtraByYoutubeId(ctx, "t429", MediaTypeMovie, 200); got == nil {
		t.Fatalf("setup failed, entry missing")
	}
	if err := RemoveAll429Rejections(); err != nil {
		t.Fatalf("RemoveAll429Rejections failed: %v", err)
	}
	if got, _ := GetExtraByYoutubeId(ctx, "t429", MediaTypeMovie, 200); got != nil {
		t.Fatalf("expected entry removed by RemoveAll429Rejections, still present: %v", got)
	}
}

func TestSetAndMarkStatusTransitions(t *testing.T) {
	ctx := context.Background()
	_ = GetStoreClient().Del(ctx, ExtrasStoreKey)
	// Set rejected persistently
	if err := SetExtraRejectedPersistent(MediaTypeMovie, 300, "Trailers", "TT", "tstate", "user"); err != nil {
		t.Fatalf("SetExtraRejectedPersistent failed: %v", err)
	}
	got, _ := GetExtraByYoutubeId(ctx, "tstate", MediaTypeMovie, 300)
	if got == nil || got.Status != "rejected" {
		t.Fatalf("expected rejected after SetExtraRejectedPersistent: %v", got)
	}

	// Mark downloaded
	if err := MarkExtraDownloaded(MediaTypeMovie, 300, "Trailers", "TT", "tstate"); err != nil {
		t.Fatalf("MarkExtraDownloaded failed: %v", err)
	}
	got2, _ := GetExtraByYoutubeId(ctx, "tstate", MediaTypeMovie, 300)
	if got2 == nil || got2.Status != "downloaded" {
		t.Fatalf("expected downloaded after MarkExtraDownloaded: %v", got2)
	}

	// Mark deleted
	if err := MarkExtraDeleted(MediaTypeMovie, 300, "Trailers", "TT", "tstate"); err != nil {
		t.Fatalf("MarkExtraDeleted failed: %v", err)
	}
	got3, _ := GetExtraByYoutubeId(ctx, "tstate", MediaTypeMovie, 300)
	if got3 == nil || got3.Status != "deleted" {
		t.Fatalf("expected deleted after MarkExtraDeleted: %v", got3)
	}
}
