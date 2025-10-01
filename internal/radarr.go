package internal

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

// ListServerFoldersHandler handles GET /api/files/list and returns subfolders for a given path
func ListServerFoldersHandler(c *gin.Context) {
	// Only allow browsing from allowed roots
	allowedRoots := []string{"/mnt", TrailarrRoot}
	reqPath := c.Query("path")
	if reqPath == "" {
		// If no path, return allowed roots
		c.JSON(200, gin.H{"folders": allowedRoots})
		return
	}
	// Security: ensure reqPath is under allowed roots
	valid := false
	for _, root := range allowedRoots {
		if reqPath == root || (len(reqPath) > len(root) && reqPath[:len(root)] == root) {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(400, gin.H{"error": "Invalid path"})
		return
	}
	// List subfolders
	entries, err := os.ReadDir(reqPath)
	if err != nil {
		c.JSON(500, gin.H{"error": err.Error()})
		return
	}
	var folders []string
	for _, entry := range entries {
		if entry.IsDir() {
			folders = append(folders, filepath.Join(reqPath, entry.Name()))
		}
	}
	c.JSON(200, gin.H{"folders": folders})
}

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

func getRadarrSettingsHandler(c *gin.Context) {
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
	radarrSection, _ := config["radarr"].(map[string]interface{})
	// Convert PathMappings keys to lowercase for frontend compatibility
	var mappings []map[string]string
	mappingSet := map[string]bool{}
	var pathMappings []map[string]interface{}
	if radarrSection != nil {
		if pm, ok := radarrSection["pathMappings"].([]interface{}); ok {
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

	// Add any root folder from Radarr API response to settings if missing
	folders, err := FetchRootFolders(radarrSection["url"].(string), radarrSection["apiKey"].(string))
	fmt.Printf("[DEBUG] Radarr root folders response: %+v\n", folders)
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
		// Save only Radarr section
		radarrSection["pathMappings"] = pathMappings
		config["radarr"] = radarrSection
		out, _ := yaml.Marshal(config)
		err := os.WriteFile(ConfigPath, out, 0644)
		if err != nil {
			fmt.Printf("[ERROR] Failed to save updated config: %v\n", err)
		} else {
			fmt.Printf("[INFO] Updated config with new root folders\n")
		}
	}
	fmt.Printf("[DEBUG] Returning Radarr settings: url=%v, apiKey=%v, pathMappings=%+v\n", radarrSection["url"], radarrSection["apiKey"], mappings)
	c.JSON(http.StatusOK, gin.H{"url": radarrSection["url"], "apiKey": radarrSection["apiKey"], "pathMappings": mappings})
}

func saveRadarrSettingsHandler(c *gin.Context) {
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
	// Update only radarr section
	radarr := map[string]interface{}{
		"url":          req.URL,
		"apiKey":       req.APIKey,
		"pathMappings": req.PathMappings,
	}
	config["radarr"] = radarr
	out, _ := yaml.Marshal(config)
	err := os.WriteFile(ConfigPath, out, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// Sync queue item and status for Radarr

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
	syncRadarrStatus.NextExecution = item.Ended.Add(15 * time.Minute)
	if len(syncRadarrStatus.Queue) > 10 {
		syncRadarrStatus.Queue = syncRadarrStatus.Queue[len(syncRadarrStatus.Queue)-10:]
	}
}

// Background sync for Radarr
func BackgroundSyncRadarr() {
	BackgroundSync(
		15*time.Minute,
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
			syncRadarrStatus.NextExecution = ended.Add(15 * time.Minute)
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
		c.JSON(http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          "Sync with Radarr",
				"interval":      "15 minutes",
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

// Lists movies without any downloaded trailer extra
func GetMoviesWithoutTrailerExtraHandler(c *gin.Context) {
	cachePath := MoviesCachePath
	movies, err := loadCache(cachePath)
	if err != nil {
		c.JSON(500, gin.H{"error": "Movie cache not found"})
		return
	}
	// Load Radarr settings to get pathMappings
	data, err := os.ReadFile(ConfigPath)
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	radarrSection, _ := config["radarr"].(map[string]interface{})
	var trailerPaths []string
	if radarrSection != nil {
		if pm, ok := radarrSection["pathMappings"].([]interface{}); ok {
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
		trailerPaths = append(trailerPaths, "/mnt/unionfs/Media/Movies")
	}
	trailerSet := findMediaWithTrailers(trailerPaths...)
	var result []map[string]interface{}
	for _, m := range movies {
		path, ok := m["path"].(string)
		if !ok || trailerSet[path] {
			continue
		}
		result = append(result, m)
	}
	c.JSON(200, gin.H{"movies": result})
}

// Example usage for Radarr
func getRadarrRootFoldersHandler(c *gin.Context) {
	// Load Radarr settings
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Config not found"})
		return
	}
	var allSettings struct {
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Config parse error"})
		return
	}
	folders, err := FetchRootFolders(allSettings.Radarr.URL, allSettings.Radarr.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"rootFolders": folders})
}
