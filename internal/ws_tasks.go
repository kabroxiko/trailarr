package internal

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var taskStatusClientsMu sync.Mutex
var taskStatusClients = make(map[*websocket.Conn]struct{})

func getWebSocketUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
}

func addTaskStatusClient(conn *websocket.Conn) {
	taskStatusClientsMu.Lock()
	taskStatusClients[conn] = struct{}{}
	taskStatusClientsMu.Unlock()
	// Send initial status
	go sendCurrentTaskStatus(conn)
}

func removeTaskStatusClient(conn *websocket.Conn) {
	taskStatusClientsMu.Lock()
	delete(taskStatusClients, conn)
	taskStatusClientsMu.Unlock()
}

func broadcastTaskStatus(status interface{}) {
	taskStatusClientsMu.Lock()
	for conn := range taskStatusClients {
		go sendTaskStatus(conn, status)
	}
	taskStatusClientsMu.Unlock()
}

func sendCurrentTaskStatus(conn *websocket.Conn) {
	status := getCurrentTaskStatus()
	sendTaskStatus(conn, status)
}

func sendTaskStatus(conn *websocket.Conn, status interface{}) {
	data, err := json.Marshal(status)
	if err != nil {
		return
	}
	conn.WriteMessage(websocket.TextMessage, data)
}

// Helper to get the same status as /api/tasks/status
func getCurrentTaskStatus() interface{} {
	// Use the same logic as GetAllTasksStatus
	var radarrStatus, sonarrStatus, extrasStatus string
	if radarrTaskStarted {
		radarrStatus = "running"
	} else {
		radarrStatus = "idle"
	}
	if sonarrTaskStarted {
		sonarrStatus = "running"
	} else {
		sonarrStatus = "idle"
	}
	if extrasTaskStarted {
		extrasStatus = "running"
	} else {
		extrasStatus = "idle"
	}
	times := GlobalTaskTimes
	calcNext := func(lastExecution time.Time, interval int) time.Time {
		if lastExecution.IsZero() {
			return time.Now().Add(time.Duration(interval) * time.Minute)
		}
		return lastExecution.Add(time.Duration(interval) * time.Minute)
	}
	schedules := []map[string]interface{}{
		{
			"name":          times.Radarr.Name,
			"interval":      times.Radarr.Interval,
			"lastExecution": times.Radarr.LastExecution,
			"lastDuration":  times.Radarr.LastDuration,
			"nextExecution": calcNext(times.Radarr.LastExecution, times.Radarr.Interval),
			"status":        radarrStatus,
		},
		{
			"name":          times.Sonarr.Name,
			"interval":      times.Sonarr.Interval,
			"lastExecution": times.Sonarr.LastExecution,
			"lastDuration":  times.Sonarr.LastDuration,
			"nextExecution": calcNext(times.Sonarr.LastExecution, times.Sonarr.Interval),
			"status":        sonarrStatus,
		},
		{
			"name":          times.Extras.Name,
			"interval":      times.Extras.Interval,
			"lastExecution": times.Extras.LastExecution,
			"lastDuration":  times.Extras.LastDuration,
			"nextExecution": calcNext(times.Extras.LastExecution, times.Extras.Interval),
			"status":        extrasStatus,
		},
	}
	queues := buildTaskQueues()
	sortTaskQueuesByQueuedDesc(queues)
	return map[string]interface{}{
		"schedules": schedules,
		"queues":    queues,
	}
}
