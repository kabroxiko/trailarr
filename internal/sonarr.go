package internal

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// import extras.go for getRejectedExtrasForMedia

func getSonarrHandler(c *gin.Context) {
	cachePath := TrailarrRoot + "/series.json"
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

	seriesPath, err := FindMediaPathByID(TrailarrRoot+"/series.json", idStr)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Series cache not found"})
		return
	}

	// Mark downloaded extras using shared logic
	MarkDownloadedExtras(results, seriesPath, "type", "title")

	// Add rejected extras from rejected_extras.json
	rejectedExtras := GetRejectedExtrasForMedia("tv", id)
	for _, rej := range rejectedExtras {
		found := false
		for _, e := range results {
			if e["url"] == rej.URL {
				// Always set status: rejected if this is a rejected extra
				e["status"] = "rejected"
				found = true
				break
			}
		}
		if !found {
			results = append(results, map[string]string{
				"type":   rej.ExtraType,
				"title":  rej.ExtraTitle,
				"url":    rej.URL,
				"status": "rejected",
			})
		}
	}

	c.JSON(http.StatusOK, gin.H{"extras": results})
}

// SyncSonarrQueueItem tracks a Sonarr sync operation
type SyncSonarrQueueItem struct {
	TaskName string
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
func SyncSonarr() {
	if !GetAutoDownloadExtras() {
		log.Println("[FORCE] Auto download of extras is disabled by general settings. Skipping forced Sonarr sync.")
		return
	}
	// Use generic ForceSyncMedia from media.go
	// Only use GlobalSyncQueue for persistence and display
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
		if item.TaskName == "sonarr" {
			syncSonarrStatus.Queue = append(syncSonarrStatus.Queue, SyncSonarrQueueItem(item))
		}
	}
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
	err := SyncMediaCacheJson("sonarr", "/api/v3/series", TrailarrRoot+"/series.json", func(m map[string]interface{}) bool {
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
	series, err := loadCache(TrailarrRoot + "/series.json")
	if err != nil {
		return err
	}
	CacheMediaPosters(
		"sonarr",
		MediaCoverPath+"/Series",
		series,
		"id",
		[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
		true, // debug
	)
	return nil
}
