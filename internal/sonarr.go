package internal

import (
	"net/http"
	"os"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/gin-gonic/gin"
)

var getSonarrPosterHandler = getImageHandler("sonarr", "serieId", "/poster-500.jpg")

var getSonarrBannerHandler = getImageHandler("sonarr", "serieId", "/fanart-1280.jpg")

func getSonarrSeriesHandler(c *gin.Context) {
	cachePath := SeriesCachePath
	series, err := loadCache(cachePath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Series cache not found"})
		return
	}
	c.JSON(200, gin.H{"series": series})
}

func getSonarrSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": allSettings.Sonarr.URL, "apiKey": allSettings.Sonarr.APIKey})
}

func saveSonarrSettingsHandler(c *gin.Context) {
	var req struct {
		URL    string `yaml:"url"`
		APIKey string `yaml:"apiKey"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	data, _ := os.ReadFile(ConfigPath)
	_ = yaml.Unmarshal(data, &allSettings)
	allSettings.Sonarr.URL = req.URL
	allSettings.Sonarr.APIKey = req.APIKey
	out, _ := yaml.Marshal(allSettings)
	err := os.WriteFile(ConfigPath, out, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// Sync queue item and status for Sonarr

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
	println("[FORCE] Executing Sync Sonarr...")
	item := SyncSonarrQueueItem{
		Queued: time.Now(),
		Status: "queued",
	}
	syncSonarrStatus.Queue = append(syncSonarrStatus.Queue, item)
	item.Started = time.Now()
	item.Status = "running"
	err := SyncSonarrImages()
	item.Ended = time.Now()
	item.Duration = item.Ended.Sub(item.Started)
	if err == nil {
		item.Status = "done"
	} else {
		item.Status = "error"
	}
	if err != nil {
		item.Error = err.Error()
		item.Status = "error"
		syncSonarrStatus.LastError = err.Error()
		println("[FORCE] Sync Sonarr error:", err.Error())
	} else {
		println("[FORCE] Sync Sonarr completed successfully.")
	}
	syncSonarrStatus.LastExecution = item.Ended
	syncSonarrStatus.LastDuration = item.Duration
	syncSonarrStatus.NextExecution = item.Ended.Add(15 * time.Minute)
	if len(syncSonarrStatus.Queue) > 10 {
		syncSonarrStatus.Queue = syncSonarrStatus.Queue[len(syncSonarrStatus.Queue)-10:]
	}
}

// Background sync for Sonarr
func BackgroundSyncSonarr() {
	BackgroundSync(
		15*time.Minute,
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
			syncSonarrStatus.NextExecution = ended.Add(15 * time.Minute)
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
		c.JSON(http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          "Sync with Sonarr",
				"interval":      "15 minutes",
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
	return SyncMediaCacheJson("sonarr", "/api/v3/series", SeriesCachePath, func(m map[string]interface{}) bool {
		stats, ok := m["statistics"].(map[string]interface{})
		if !ok {
			return false
		}
		episodeFileCount, ok := stats["episodeFileCount"].(float64)
		return ok && episodeFileCount >= 1
	})
}
