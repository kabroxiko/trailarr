package internal

import (
	"context"
	"encoding/json"
	"os"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var downloadQueueClientsMu sync.Mutex
var downloadQueueClients = make(map[*websocket.Conn]struct{})

// AddDownloadQueueClient adds a WebSocket client to the set
func AddDownloadQueueClient(conn *websocket.Conn) {
	downloadQueueClientsMu.Lock()
	downloadQueueClients[conn] = struct{}{}
	downloadQueueClientsMu.Unlock()
	go SendCurrentDownloadQueue(conn)
}

// RemoveDownloadQueueClient removes a WebSocket client from the set
func RemoveDownloadQueueClient(conn *websocket.Conn) {
	downloadQueueClientsMu.Lock()
	delete(downloadQueueClients, conn)
	downloadQueueClientsMu.Unlock()
}

// BroadcastDownloadQueueChanges sends only the changed queue items to all clients
func BroadcastDownloadQueueChanges(changed []DownloadQueueItem) {
	if len(changed) == 0 {
		return
	}
	data, err := json.Marshal(map[string]interface{}{
		"type":  "download_queue_update",
		"queue": changed,
	})
	if err != nil {
		TrailarrLog(DEBUG, "WebSocket", "Failed to marshal queue change: %v", err)
		return
	}
	downloadQueueClientsMu.Lock()
	count := 0
	for conn := range downloadQueueClients {
		err := conn.WriteMessage(websocket.TextMessage, data)
		if err != nil {
			TrailarrLog(DEBUG, "WebSocket", "Failed to send message to client: %v", err)
		} else {
			count++
		}
	}
	TrailarrLog(INFO, "WebSocket", "Broadcasted download_queue_update (changes only) to %d clients. Message: %s", count, string(data))
	downloadQueueClientsMu.Unlock()
}

// SendCurrentDownloadQueue sends the current queue to a single client
func SendCurrentDownloadQueue(conn *websocket.Conn) {
	queue := GetCurrentDownloadQueue()
	data, err := json.Marshal(map[string]interface{}{
		"type":  "download_queue_update",
		"queue": queue,
	})
	if err != nil {
		return
	}
	conn.WriteMessage(websocket.TextMessage, data)
}

// GetCurrentDownloadQueue loads the current download queue from file

func GetCurrentDownloadQueue() []DownloadQueueItem {
	ctx := context.Background()
	client := GetStoreClient()
	var queue []DownloadQueueItem
	items, err := client.LRange(ctx, DownloadQueue, 0, -1)
	if err == nil {
		for _, itemStr := range items {
			var item DownloadQueueItem
			if err := json.Unmarshal([]byte(itemStr), &item); err == nil {
				queue = append(queue, item)
			}
		}
	}
	return queue
}

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
		time.Sleep(DownloadQueueWatcherInterval)
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
