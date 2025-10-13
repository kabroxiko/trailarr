package internal

import (
	"encoding/json"
	"os"
	"time"
)

// DownloadQueueWatcher watches download_queue.json for changes and triggers a callback
func DownloadQueueWatcher(path string, onChange func(changed []DownloadQueueItem)) {
	var lastModTime time.Time
	var lastQueue []DownloadQueueItem
	for {
		info, err := os.Stat(path)
		if err == nil {
			if info.ModTime() != lastModTime {
				lastModTime = info.ModTime()
				// Read new queue
				var newQueue []DownloadQueueItem
				data, err := os.ReadFile(path)
				if err == nil {
					_ = json.Unmarshal(data, &newQueue)
				}
				// Deduplicate by YouTubeID, keep latest by QueuedAt
				newQueue = DedupLatestByYouTubeID(newQueue)
				lastQueue = DedupLatestByYouTubeID(lastQueue)
				// Compute diff
				changed := DiffDownloadQueue(lastQueue, newQueue)
				if len(changed) > 0 {
					onChange(changed)
				}
				lastQueue = newQueue
			}
		}
		time.Sleep(1 * time.Second)
	}
}

// DedupLatestByYouTubeID keeps only the latest item per YouTubeID (by QueuedAt)
func DedupLatestByYouTubeID(queue []DownloadQueueItem) []DownloadQueueItem {
	m := make(map[string]DownloadQueueItem)
	for _, item := range queue {
		prev, ok := m[item.YouTubeID]
		if !ok || item.QueuedAt.After(prev.QueuedAt) {
			m[item.YouTubeID] = item
		}
	}
	out := make([]DownloadQueueItem, 0, len(m))
	for _, v := range m {
		out = append(out, v)
	}
	return out
}

// DiffDownloadQueue returns only the changed or new items (by YouTubeID and Status)
func DiffDownloadQueue(oldQ, newQ []DownloadQueueItem) []DownloadQueueItem {
	oldMap := make(map[string]DownloadQueueItem)
	for _, item := range oldQ {
		oldMap[item.YouTubeID] = item
	}
	var changed []DownloadQueueItem
	for _, item := range newQ {
		old, ok := oldMap[item.YouTubeID]
		if !ok || old.Status != item.Status {
			changed = append(changed, item)
		}
	}
	return changed
}
