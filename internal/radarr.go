package internal

import (
	"time"
)

// ...existing code...

// Use shared SyncStatus from media.go
var syncRadarrStatus = NewSyncStatus()

// Handler to force sync Radarr
func SyncRadarr() {
	SyncMedia(
		"radarr",
		SyncRadarrImages,
		Timings,
		&syncRadarrStatus.LastError,
		&syncRadarrStatus.LastExecution,
		&syncRadarrStatus.LastDuration,
		&syncRadarrStatus.NextExecution,
	)
	syncRadarrStatus.Queue = nil
	for _, item := range GlobalSyncQueue {
		if item.TaskName == "radarr" {
			syncRadarrStatus.Queue = append(syncRadarrStatus.Queue, item)
		}
	}
}

// Exported Radarr status getters for main.go
func RadarrLastExecution() time.Time    { return LastExecution(syncRadarrStatus) }
func RadarrLastDuration() time.Duration { return LastDuration(syncRadarrStatus) }
func RadarrNextExecution() time.Time    { return NextExecution(syncRadarrStatus) }
func RadarrLastError() string           { return LastError(syncRadarrStatus) }
func RadarrQueue() []SyncQueueItem      { return Queue(syncRadarrStatus) }

// Exported handler for Radarr status
var GetRadarrStatusHandler = GetSyncStatusHandler("radarr", syncRadarrStatus, "Radarr")

func SyncRadarrImages() error {
	return SyncMediaImages(
		"radarr",
		"/api/v3/movie",
		TrailarrRoot+"/movies.json",
		func(m map[string]interface{}) bool {
			hasFile, ok := m["hasFile"].(bool)
			return ok && hasFile
		},
		MediaCoverPath+"/Movies",
		[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
	)
}
