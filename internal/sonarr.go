package internal

import (
	"time"
)

// Use shared SyncStatus from media.go
var syncSonarrStatus = NewSyncStatus()

// Handler to force sync Sonarr
func SyncSonarr() {
	SyncMedia(
		"sonarr",
		SyncSonarrImages,
		Timings,
		&syncSonarrStatus.LastError,
		&syncSonarrStatus.LastExecution,
		&syncSonarrStatus.LastDuration,
		&syncSonarrStatus.NextExecution,
	)
	syncSonarrStatus.Queue = nil
	for _, item := range GlobalSyncQueue {
		if item.TaskId == "sonarr" {
			syncSonarrStatus.Queue = append(syncSonarrStatus.Queue, item)
		}
	}
}

// Exported Sonarr status getters for main.go
func SonarrLastExecution() time.Time    { return LastExecution(syncSonarrStatus) }
func SonarrLastDuration() time.Duration { return LastDuration(syncSonarrStatus) }
func SonarrNextExecution() time.Time    { return NextExecution(syncSonarrStatus) }
func SonarrLastError() string           { return LastError(syncSonarrStatus) }
func SonarrQueue() []SyncQueueItem      { return Queue(syncSonarrStatus) }

// Exported handler for Sonarr status
var GetSonarrStatusHandler = GetSyncStatusHandler("sonarr", syncSonarrStatus, "Sonarr")

func SyncSonarrImages() error {
	return SyncMediaImages(
		"sonarr",
		"/api/v3/series",
		TrailarrRoot+"/series.json",
		func(m map[string]interface{}) bool {
			stats, ok := m["statistics"].(map[string]interface{})
			if !ok {
				return false
			}
			episodeFileCount, ok := stats["episodeFileCount"].(float64)
			return ok && episodeFileCount >= 1
		},
		MediaCoverPath+"/Series",
		[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
	)
}
