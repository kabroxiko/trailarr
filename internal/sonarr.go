package internal

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	// adjust to actual module path if needed
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

func getSonarrSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": "", "pathMappings": []interface{}{}})
		return
	}
	sonarrSection, _ := config["sonarr"].(map[string]interface{})
	// Convert PathMappings keys to lowercase for frontend compatibility
	var mappings []map[string]string
	mappingSet := map[string]bool{}
	var pathMappings []map[string]interface{}
	if sonarrSection != nil {
		if pm, ok := sonarrSection["pathMappings"].([]interface{}); ok {
			for _, m := range pm {
				if mMap, ok := m.(map[string]interface{}); ok {
					from := ""
					to := ""
					if v, ok := mMap["from"].(string); ok {
						from = v
					}
					if v, ok := mMap["to"].(string); ok {
						to = v
					}
					mappings = append(mappings, map[string]string{"from": from, "to": to})
					mappingSet[from] = true
					pathMappings = append(pathMappings, map[string]interface{}{"from": from, "to": to})
				}
			}
		}
	}

	// Add any root folder from Sonarr API response to settings if missing
	folders, err := FetchRootFolders(sonarrSection["url"].(string), sonarrSection["apiKey"].(string))
	fmt.Printf("[DEBUG] Sonarr root folders response: %+v\n", folders)
	updated := false
	for _, f := range folders {
		if path, ok := f["path"].(string); ok {
			if !mappingSet[path] {
				fmt.Printf("[INFO] Adding missing root folder to settings: %s\n", path)
				pathMappings = append(pathMappings, map[string]interface{}{"from": path, "to": ""})
				mappings = append(mappings, map[string]string{"from": path, "to": ""})
				updated = true
			}
		}
	}
	if updated {
		// Save only Sonarr section
		sonarrSection["pathMappings"] = pathMappings
		config["sonarr"] = sonarrSection
		out, _ := yaml.Marshal(config)
		err := os.WriteFile(ConfigPath, out, 0644)
		if err != nil {
			fmt.Printf("[ERROR] Failed to save updated config: %v\n", err)
		} else {
			fmt.Printf("[INFO] Updated config with new root folders\n")
		}
	}
	fmt.Printf("[DEBUG] Returning Sonarr settings: url=%v, apiKey=%v, pathMappings=%+v\n", sonarrSection["url"], sonarrSection["apiKey"], mappings)
	c.JSON(http.StatusOK, gin.H{"url": sonarrSection["url"], "apiKey": sonarrSection["apiKey"], "pathMappings": mappings})
}

func saveSonarrSettingsHandler(c *gin.Context) {
	var req struct {
		URL          string `yaml:"url"`
		APIKey       string `yaml:"apiKey"`
		PathMappings []struct {
			From string `yaml:"from"`
			To   string `yaml:"to"`
		} `yaml:"pathMappings"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	// Read existing config as map[string]interface{} to preserve all keys
	data, _ := os.ReadFile(ConfigPath)
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	// Update only sonarr section
	sonarr := map[string]interface{}{
		"url":          req.URL,
		"apiKey":       req.APIKey,
		"pathMappings": req.PathMappings,
	}
	config["sonarr"] = sonarr
	out, _ := yaml.Marshal(config)
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

// Lists series without any downloaded trailer extra
func GetSeriesWithoutTrailerExtraHandler(c *gin.Context) {
	cachePath := SeriesCachePath
	series, err := loadCache(cachePath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Series cache not found"})
		return
	}
	// Load Sonarr settings to get pathMappings
	data, err := os.ReadFile(ConfigPath)
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	sonarrSection, _ := config["sonarr"].(map[string]interface{})
	var trailerPaths []string
	if sonarrSection != nil {
		if pm, ok := sonarrSection["pathMappings"].([]interface{}); ok {
			for _, m := range pm {
				if mMap, ok := m.(map[string]interface{}); ok {
					if to, ok := mMap["to"].(string); ok && to != "" {
						trailerPaths = append(trailerPaths, to)
					}
				}
			}
		}
	}
	if len(trailerPaths) == 0 {
		// fallback to default if no mappings
		trailerPaths = append(trailerPaths, "/mnt/unionfs/Media/TV")
	}
	trailerSet := findMediaWithTrailers(trailerPaths...)
	var result []map[string]interface{}
	for _, s := range series {
		path, ok := s["path"].(string)
		if !ok || trailerSet[path] {
			continue
		}
		result = append(result, s)
	}
	c.JSON(200, gin.H{"series": result})
}

// Example usage for Sonarr
func getSonarrRootFoldersHandler(c *gin.Context) {
	// Load Sonarr settings
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Config not found"})
		return
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Config parse error"})
		return
	}
	folders, err := FetchRootFolders(allSettings.Sonarr.URL, allSettings.Sonarr.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rootFolders": folders})
}
