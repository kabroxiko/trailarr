package internal

import (
	"testing"
	"time"
)

func makeItem(yid, status string, t time.Time) DownloadQueueItem {
	return DownloadQueueItem{YouTubeID: yid, Status: status, QueuedAt: t}
}

func TestDedupLatestByYouTubeID(t *testing.T) {
	now := time.Now()
	q := []DownloadQueueItem{
		makeItem("a", "queued", now.Add(-2*time.Minute)),
		makeItem("b", "queued", now.Add(-1*time.Minute)),
		makeItem("a", "downloading", now),
	}
	got := DedupLatestByYouTubeID(q)
	if len(got) != 2 {
		t.Fatalf("expected 2 items after dedup, got %d", len(got))
	}
	// ensure 'a' has latest status
	for _, it := range got {
		if it.YouTubeID == "a" && it.Status != "downloading" {
			t.Fatalf("expected latest status for a to be downloading, got %s", it.Status)
		}
	}
}

func TestDiffDownloadQueue(t *testing.T) {
	now := time.Now()
	oldQ := []DownloadQueueItem{makeItem("a", "queued", now), makeItem("b", "queued", now)}
	newQ := []DownloadQueueItem{makeItem("a", "downloading", now), makeItem("c", "queued", now)}
	got := DiffDownloadQueue(oldQ, newQ)
	if len(got) != 2 {
		t.Fatalf("expected 2 changed items (a status changed, c new), got %d", len(got))
	}
}

func TestFindLastQueueStatus(t *testing.T) {
	now := time.Now()
	q := []DownloadQueueItem{
		makeItem("x", "queued", now.Add(-10*time.Minute)),
		makeItem("y", "queued", now.Add(-5*time.Minute)),
		makeItem("x", "downloading", now),
	}
	st := findLastQueueStatus(q, "x")
	if st == nil || st.Status != "downloading" {
		t.Fatalf("expected last status for x to be downloading, got %+v", st)
	}
	if st2 := findLastQueueStatus(q, "z"); st2 != nil {
		t.Fatalf("expected nil for unknown id z, got %+v", st2)
	}
}
