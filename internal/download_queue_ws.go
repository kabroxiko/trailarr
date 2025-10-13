package internal

import (
	"encoding/json"
	"sync"

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
	var queue []DownloadQueueItem
	_ = ReadJSONFile(DownloadQueuePath, &queue)
	return queue
}
