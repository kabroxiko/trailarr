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
	cachePath := MoviesCachePath
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

	moviePath, err := FindMediaPathByID(MoviesCachePath, idStr)
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
	println("[FORCE] Executing Sync Radarr...")
	item := SyncRadarrQueueItem{
		Queued: time.Now(),
		Status: "queued",
	}
	syncRadarrStatus.Queue = append(syncRadarrStatus.Queue, item)
	item.Started = time.Now()
	item.Status = "running"
	err := SyncRadarrImages()
	item.Ended = time.Now()
	item.Duration = item.Ended.Sub(item.Started)
	item.Status = "done"
	if err != nil {
		item.Error = err.Error()
		item.Status = "error"
		syncRadarrStatus.LastError = err.Error()
		println("[FORCE] Sync Radarr error:", err.Error())
	} else {
		println("[FORCE] Sync Radarr completed successfully.")
	}
	syncRadarrStatus.LastExecution = item.Ended
	syncRadarrStatus.LastDuration = item.Duration
	interval := Timings["radarr"]
	syncRadarrStatus.NextExecution = item.Ended.Add(time.Duration(interval) * time.Minute)
	if len(syncRadarrStatus.Queue) > 10 {
		syncRadarrStatus.Queue = syncRadarrStatus.Queue[len(syncRadarrStatus.Queue)-10:]
	}
}

// Background sync for Radarr
func BackgroundSyncRadarr() {
	interval := Timings["radarr"]
	BackgroundSync(
		time.Duration(interval)*time.Minute,
		SyncRadarrImages,
		func(item interface{}) {
			syncRadarrStatus.Queue = append(syncRadarrStatus.Queue, *item.(*SyncRadarrQueueItem))
		},
		func() interface{} {
			return &SyncRadarrQueueItem{Queued: time.Now(), Status: "queued"}
		},
		func(item interface{}, started, ended time.Time, duration time.Duration, status, errStr string) {
			i := item.(*SyncRadarrQueueItem)
			i.Started = started
			i.Ended = ended
			i.Duration = duration
			i.Status = status
			i.Error = errStr
			if status == "error" {
				syncRadarrStatus.LastError = errStr
			}
			syncRadarrStatus.LastExecution = ended
			syncRadarrStatus.LastDuration = duration
			syncRadarrStatus.NextExecution = ended.Add(time.Duration(interval) * time.Minute)
		},
		func() {
			if len(syncRadarrStatus.Queue) > 10 {
				syncRadarrStatus.Queue = syncRadarrStatus.Queue[len(syncRadarrStatus.Queue)-10:]
			}
		},
	)
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
	err := SyncMediaCacheJson("radarr", "/api/v3/movie", MoviesCachePath, func(m map[string]interface{}) bool {
		hasFile, ok := m["hasFile"].(bool)
		return ok && hasFile
	})
	if err != nil {
		return err
	}
	movies, err := loadCache(MoviesCachePath)
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
