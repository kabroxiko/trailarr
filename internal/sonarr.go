package internal

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var getSonarrPosterHandler = getImageHandler("sonarr", "id", "/poster-500.jpg")

var getSonarrBannerHandler = getImageHandler("sonarr", "id", "/fanart-1280.jpg")

func getSonarrHandler(c *gin.Context) {
	cachePath := SeriesCachePath
	series, err := loadCache(cachePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Series cache not found"})
		return
	}
	// If id query param is present, filter for that series only
	idParam := c.Query("id")
	var filtered []map[string]interface{}
	if idParam != "" {
		for _, s := range series {
			if id, ok := s["id"]; ok && fmt.Sprintf("%v", id) == idParam {
				filtered = append(filtered, s)
				break
			}
		}
	} else {
		filtered = series
	}
	// Removed extras download check from list endpoint; should only be done in detail endpoint
	c.JSON(200, gin.H{"series": filtered})
}

func getSeriesExtrasHandler(c *gin.Context) {
	idStr := c.Param("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)
	results, err := SearchExtras("tv", id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	seriesPath, err := FindMediaPathByID(SeriesCachePath, idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Series cache not found"})
		return
	}

	// Mark downloaded extras using shared logic
	MarkDownloadedExtras(results, seriesPath, "type", "title")

	c.JSON(http.StatusOK, gin.H{"extras": results})
}

// SyncSonarrQueueItem tracks a Sonarr sync operation
type SyncSonarrQueueItem struct {
	Queued   time.Time
	Started  time.Time
	Ended    time.Time
	Duration time.Duration
	Status   string
	Error    string
}

// syncSonarrStatus tracks last sync status and times for Sonarr
var syncSonarrStatus = struct {
	LastExecution time.Time
	LastDuration  time.Duration
	NextExecution time.Time
	LastError     string
	Queue         []SyncSonarrQueueItem
}{
	Queue: make([]SyncSonarrQueueItem, 0),
}

// Handler to force sync Sonarr
func ForceSyncSonarr() {
	// Use generic ForceSyncMedia from media.go
	var tempQueue []SyncQueueItem
	for _, item := range syncSonarrStatus.Queue {
		tempQueue = append(tempQueue, SyncQueueItem{
			Queued:   item.Queued,
			Started:  item.Started,
			Ended:    item.Ended,
			Duration: item.Duration,
			Status:   item.Status,
			Error:    item.Error,
		})
	}
	ForceSyncMedia(
		"sonarr",
		SyncSonarrImages,
		Timings,
		&tempQueue,
		&syncSonarrStatus.LastError,
		&syncSonarrStatus.LastExecution,
		&syncSonarrStatus.LastDuration,
		&syncSonarrStatus.NextExecution,
	)
	syncSonarrStatus.Queue = nil
	for _, item := range tempQueue {
		syncSonarrStatus.Queue = append(syncSonarrStatus.Queue, SyncSonarrQueueItem{
			Queued:   item.Queued,
			Started:  item.Started,
			Ended:    item.Ended,
			Duration: item.Duration,
			Status:   item.Status,
			Error:    item.Error,
		})
	}
}

// Background sync for Sonarr
func BackgroundSyncSonarr() {
	interval := Timings["sonarr"]
	BackgroundSync(
		time.Duration(interval)*time.Minute,
		SyncSonarrImages,
		func(item interface{}) {
			syncSonarrStatus.Queue = append(syncSonarrStatus.Queue, *item.(*SyncSonarrQueueItem))
		},
		func() interface{} {
			return &SyncSonarrQueueItem{Queued: time.Now(), Status: "queued"}
		},
		func(item interface{}, started, ended time.Time, duration time.Duration, status, errStr string) {
			i := item.(*SyncSonarrQueueItem)
			i.Started = started
			i.Ended = ended
			i.Duration = duration
			i.Status = status
			i.Error = errStr
			if status == "error" {
				syncSonarrStatus.LastError = errStr
			}
			syncSonarrStatus.LastExecution = ended
			syncSonarrStatus.LastDuration = duration
			syncSonarrStatus.NextExecution = ended.Add(time.Duration(interval) * time.Minute)
		},
		func() {
			if len(syncSonarrStatus.Queue) > 10 {
				syncSonarrStatus.Queue = syncSonarrStatus.Queue[len(syncSonarrStatus.Queue)-10:]
			}
		},
	)
}

// Exported Sonarr status getters for main.go
func SonarrLastExecution() time.Time     { return syncSonarrStatus.LastExecution }
func SonarrLastDuration() time.Duration  { return syncSonarrStatus.LastDuration }
func SonarrNextExecution() time.Time     { return syncSonarrStatus.NextExecution }
func SonarrLastError() string            { return syncSonarrStatus.LastError }
func SonarrQueue() []SyncSonarrQueueItem { return syncSonarrStatus.Queue }

// Exported handler for Sonarr status
func GetSonarrStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		interval := Timings["sonarr"]
		c.JSON(http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          "Sync with Sonarr",
				"interval":      fmt.Sprintf("%d minutes", interval),
				"lastExecution": SonarrLastExecution(),
				"lastDuration":  SonarrLastDuration().String(),
				"nextExecution": SonarrNextExecution(),
				"lastError":     SonarrLastError(),
			},
			"queue": SonarrQueue(),
		})
	}
}

func SyncSonarrImages() error {
	err := SyncMediaCacheJson("sonarr", "/api/v3/series", SeriesCachePath, func(m map[string]interface{}) bool {
		stats, ok := m["statistics"].(map[string]interface{})
		if !ok {
			return false
		}
		episodeFileCount, ok := stats["episodeFileCount"].(float64)
		return ok && episodeFileCount >= 1
	})
	if err != nil {
		return err
	}
	series, err := loadCache(SeriesCachePath)
	if err != nil {
		return err
	}
	CacheMediaPosters(
		"sonarr",
		MediaCoverPath+"Series",
		series,
		"id",
		[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
		true, // debug
	)
	return nil
}
