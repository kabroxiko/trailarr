package internal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var getRadarrPosterHandler = getImageHandler("radarr", "id", "/poster-500.jpg")

var getRadarrBannerHandler = getImageHandler("radarr", "id", "/fanart-1280.jpg")

func getRadarrHandler(c *gin.Context) {
	cachePath := TrailarrRoot + "/movies.json"
	movies, err := loadCache(cachePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Movie cache not found"})
		return
	}
	// If id query param is present, filter for that movie only
	idParam := c.Query("id")
	var filtered []map[string]interface{}
	if idParam != "" {
		for _, m := range movies {
			if id, ok := m["id"]; ok && fmt.Sprintf("%v", id) == idParam {
				filtered = append(filtered, m)
				break
			}
		}
	} else {
		filtered = movies
	}
	// Removed extras download check from list endpoint; should only be done in detail endpoint
	c.JSON(http.StatusOK, gin.H{"movies": filtered})
}

func getMovieExtrasHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)
	results, err := SearchExtras("movie", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	moviePath, err := FindMediaPathByID(TrailarrRoot+"/movies.json", idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Movie cache not found"})
		return
	}

	// Mark downloaded extras using shared logic
	MarkDownloadedExtras(results, moviePath, "type", "title")

	c.JSON(http.StatusOK, gin.H{"extras": results})
}

// SyncRadarrQueueItem tracks a Radarr sync operation
type SyncRadarrQueueItem struct {
	TaskName string
	Queued   time.Time
	Started  time.Time
	Ended    time.Time
	Duration time.Duration
	Status   string
	Error    string
}

// syncRadarrStatus tracks last sync status and times for Radarr
var syncRadarrStatus = struct {
	LastExecution time.Time
	LastDuration  time.Duration
	NextExecution time.Time
	LastError     string
	Queue         []SyncRadarrQueueItem
}{
	Queue: make([]SyncRadarrQueueItem, 0),
}

// Handler to force sync Radarr
func ForceSyncRadarr() {
	if !GetAutoDownloadExtras() {
		println("[FORCE] Auto download of extras is disabled by general settings. Skipping forced Radarr sync.")
		return
	}
	// Use generic ForceSyncMedia from media.go
	// Only use GlobalSyncQueue for persistence and display
	ForceSyncMedia(
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
			syncRadarrStatus.Queue = append(syncRadarrStatus.Queue, SyncRadarrQueueItem{
				TaskName: item.TaskName,
				Queued:   item.Queued,
				Started:  item.Started,
				Ended:    item.Ended,
				Duration: item.Duration,
				Status:   item.Status,
				Error:    item.Error,
			})
		}
	}
}

// Exported Radarr status getters for main.go
func RadarrLastExecution() time.Time     { return syncRadarrStatus.LastExecution }
func RadarrLastDuration() time.Duration  { return syncRadarrStatus.LastDuration }
func RadarrNextExecution() time.Time     { return syncRadarrStatus.NextExecution }
func RadarrLastError() string            { return syncRadarrStatus.LastError }
func RadarrQueue() []SyncRadarrQueueItem { return syncRadarrStatus.Queue }

// Exported handler for Radarr status
func GetRadarrStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		interval := Timings["radarr"]
		c.JSON(http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          "Sync with Radarr",
				"interval":      fmt.Sprintf("%d minutes", interval),
				"lastExecution": RadarrLastExecution(),
				"lastDuration":  RadarrLastDuration().String(),
				"nextExecution": RadarrNextExecution(),
				"lastError":     RadarrLastError(),
			},
			"queue": RadarrQueue(),
		})
	}
}

func SyncRadarrImages() error {
	err := SyncMediaCacheJson("radarr", "/api/v3/movie", TrailarrRoot+"/movies.json", func(m map[string]interface{}) bool {
		hasFile, ok := m["hasFile"].(bool)
		return ok && hasFile
	})
	if err != nil {
		return err
	}
	movies, err := loadCache(TrailarrRoot + "/movies.json")
	if err != nil {
		return err
	}
	CacheMediaPosters(
		"radarr",
		MediaCoverPath+"Movies",
		movies,
		"id",
		[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
		true, // debug
	)
	return nil
}
