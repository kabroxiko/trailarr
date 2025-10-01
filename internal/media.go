package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var GlobalSyncQueue []SyncQueueItem

const (
	MoviesWantedFile = TrailarrRoot + "/movies_wanted.json"
	SeriesWantedFile = TrailarrRoot + "/series_wanted.json"
	queueFile        = TrailarrRoot + "/queue.json"
)

// DownloadMissingExtras downloads missing extras for a given media type ("movie" or "tv")
func DownloadMissingExtras(mediaType string, cachePath string) error {
	if !GetAutoDownloadExtras() {
		TrailarrLog("Info", "DownloadMissingExtras", "Auto download of extras is disabled by general settings.")
		return nil
	}
	TrailarrLog("Info", "DownloadMissingExtras", "DownloadMissingExtras: mediaType=%s, cachePath=%s", mediaType, cachePath)
	items, err := loadCache(cachePath)
	if err != nil {
		TrailarrLog("Warn", "DownloadMissingExtras", "Failed to load cache: %v", err)
		return err
	}
	for _, m := range items {
		idInt, ok := parseMediaID(m["id"])
		if !ok {
			TrailarrLog("Warn", "DownloadMissingExtras", "Missing or invalid id in item: %v", m)
			continue
		}
		TrailarrLog("Debug", "DownloadMissingExtras", "Item idInt=%d", idInt)
		extras, err := SearchExtras(mediaType, idInt)
		if err != nil {
			TrailarrLog("Warn", "DownloadMissingExtras", "SearchExtras error: %v", err)
			continue
		}
		mediaPath, err := FindMediaPathByID(cachePath, fmt.Sprintf("%v", m["id"]))
		if err != nil || mediaPath == "" {
			TrailarrLog("Warn", "DownloadMissingExtras", "FindMediaPathByID error or empty: %v, mediaPath=%s", err, mediaPath)
			continue
		}
		TrailarrLog("Debug", "DownloadMissingExtras", "mediaPath=%s", mediaPath)
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		config, _ := GetExtraTypesConfig()
		filterAndDownloadExtras(mediaType, mediaPath, extras, config)
	}
	return nil
}

func parseMediaID(id interface{}) (int, bool) {
	var idInt int
	switch v := id.(type) {
	case int:
		idInt = v
	case float64:
		idInt = int(v)
	case string:
		_, err := fmt.Sscanf(v, "%d", &idInt)
		if err != nil {
			return 0, false
		}
	default:
		return 0, false
	}
	return idInt, true
}

func filterAndDownloadExtras(mediaType, mediaPath string, extras []map[string]string, config ExtraTypesConfig) {
	for _, extra := range extras {
		if shouldDownloadExtra(extra, config) {
			err := handleExtraDownload(mediaType, mediaPath, extra)
			if err != nil {
				TrailarrLog("Warn", "DownloadMissingExtras", "Failed to download: %v", err)
			}
		}
	}
}

func shouldDownloadExtra(extra map[string]string, config ExtraTypesConfig) bool {
	if extra["downloaded"] != "false" || extra["url"] == "" {
		return false
	}
	typeName := extra["type"]
	canonical := canonicalizeExtraType(typeName, "")
	return isExtraTypeEnabled(config, canonical)
}

func handleExtraDownload(mediaType, mediaPath string, extra map[string]string) error {
	_, err := DownloadYouTubeExtra(mediaType, filepath.Base(mediaPath), extra["type"], extra["title"], extra["url"])
	return err
}

// DownloadMissingMoviesExtras downloads missing extras for all movies
func DownloadMissingMoviesExtras() error {
	return DownloadMissingExtras("movie", MoviesWantedFile)
}

// DownloadMissingSeriesExtras downloads missing extras for all series
func DownloadMissingSeriesExtras() error {
	return DownloadMissingExtras("tv", SeriesWantedFile)
}

// Parametric force sync for Radarr/Sonarr
type SyncQueueItem struct {
	TaskName string
	Queued   time.Time
	Started  time.Time
	Ended    time.Time
	Duration time.Duration
	Status   string
	Error    string
}

func saveQueue() {
	// Only save if queue is non-empty
	if len(GlobalSyncQueue) == 0 {
		return
	}
	f, err := os.Create(queueFile)
	if err != nil {
		return
	}
	defer f.Close()
	// Convert zero time fields to nil for JSON output
	type queueItemOut struct {
		TaskName string         `json:"TaskName"`
		Queued   time.Time      `json:"Queued"`
		Started  *time.Time     `json:"Started"`
		Ended    *time.Time     `json:"Ended"`
		Duration *time.Duration `json:"Duration"`
		Status   string         `json:"Status"`
		Error    string         `json:"Error"`
	}
	out := make([]queueItemOut, 0, len(GlobalSyncQueue))
	for _, item := range GlobalSyncQueue {
		var startedPtr, endedPtr *time.Time
		var durationPtr *time.Duration
		if !item.Started.IsZero() {
			startedPtr = &item.Started
			if !item.Ended.IsZero() {
				endedPtr = &item.Ended
			}
			// Duration is only valid if Started and Ended are set
			if endedPtr != nil && item.Duration > 0 {
				durationPtr = &item.Duration
			}
			out = append(out, queueItemOut{
				TaskName: item.TaskName,
				Queued:   item.Queued,
				Started:  startedPtr,
				Ended:    endedPtr,
				Duration: durationPtr,
				Status:   item.Status,
				Error:    item.Error,
			})
		}
		_ = json.NewEncoder(f).Encode(out)
	}
}

// SyncMedia executes a sync for the given section ("radarr" or "sonarr")
// syncFunc: function to perform the sync (e.g. SyncRadarrImages or SyncSonarrImages)
// timings: map of intervals (e.g. Timings)
// queue: pointer to a slice of SyncQueueItem
// lastError, lastExecution, lastDuration, nextExecution: pointers to status fields
func SyncMedia(
	section string,
	syncFunc func() error,
	timings map[string]int,
	lastError *string,
	lastExecution *time.Time,
	lastDuration *time.Duration,
	nextExecution *time.Time,
) {
	println("[FORCE] Executing Sync", section, "...")
	TrailarrLog("Info", "SyncService", "Starting sync for section: %s", section)
	// Truncate queue before adding new item to avoid idx out of range
	if len(GlobalSyncQueue) >= 10 {
		GlobalSyncQueue = GlobalSyncQueue[len(GlobalSyncQueue)-9:]
	}
	item := SyncQueueItem{
		TaskName: section,
		Queued:   time.Now(),
		Status:   "queued",
	}
	GlobalSyncQueue = append(GlobalSyncQueue, item)
	saveQueue()

	// Find the last index for the current section (radarr or sonarr)
	idx := -1
	for i := len(GlobalSyncQueue) - 1; i >= 0; i-- {
		if GlobalSyncQueue[i].TaskName == section {
			idx = i
			break
		}
	}
	if idx == -1 {
		println("[ERROR] Could not find queue item for section:", section)
		return
	}

	GlobalSyncQueue[idx].Started = time.Now()
	GlobalSyncQueue[idx].Status = "running"
	saveQueue()

	TrailarrLog("Debug", "SyncService", "Invoking syncFunc for section: %s", section)
	err := syncFunc()
	GlobalSyncQueue[idx].Ended = time.Now()
	GlobalSyncQueue[idx].Duration = GlobalSyncQueue[idx].Ended.Sub(GlobalSyncQueue[idx].Started)
	saveQueue()
	if err != nil {
		GlobalSyncQueue[idx].Error = err.Error()
		GlobalSyncQueue[idx].Status = "failed"
		saveQueue()
		TrailarrLog("Error", "SyncService", "Sync %s error: %s", section, err.Error())
	} else {
		GlobalSyncQueue[idx].Status = "success"
		saveQueue()
		TrailarrLog("Info", "SyncService", "Synced cache for %s.", section)
	}
	TrailarrLog("Info", "SyncService", "Finished sync for section: %s", section)
	*lastExecution = GlobalSyncQueue[idx].Ended
	*lastDuration = GlobalSyncQueue[idx].Duration
	interval := timings[section]
	*nextExecution = GlobalSyncQueue[idx].Ended.Add(time.Duration(interval) * time.Minute)
}

// Helper to fetch and cache poster image
func fetchAndCachePoster(localPath, posterUrl, section string) error {
	resp, err := http.Get(posterUrl)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		TrailarrLog("Warn", "CacheMediaPosters", "Failed to fetch poster image from %s", section)
		return fmt.Errorf("failed to fetch poster image from %s", section)
	}
	defer resp.Body.Close()
	out, err := os.Create(localPath)
	if err != nil {
		TrailarrLog("Warn", "CacheMediaPosters", "Failed to cache poster image for %s", section)
		return fmt.Errorf("failed to cache poster image for %s", section)
	}
	_, _ = io.Copy(out, resp.Body)
	out.Close()
	return nil
}

// Parametrized poster caching for Radarr/Sonarr
func CacheMediaPosters(
	section string, // "radarr" or "sonarr"
	baseDir string, // e.g. MediaCoverPath + "Movies" or MediaCoverPath + "Series"
	idList []map[string]interface{}, // loaded cache
	idKey string, // "id"
	posterSuffixes []string, // ["/poster-500.jpg", "/fanart-1280.jpg"]
	debug bool, // enable debug output
) {
	TrailarrLog("Info", "CacheMediaPosters", "Starting poster caching for section: %s, baseDir: %s, items: %d", section, baseDir, len(idList))
	for _, item := range idList {
		id := fmt.Sprintf("%v", item[idKey])
		for _, suffix := range posterSuffixes {
			idDir := baseDir + "/" + id
			if err := os.MkdirAll(idDir, 0775); err != nil {
				TrailarrLog("Warn", "CacheMediaPosters", "Failed to create dir %s: %v", idDir, err)
				continue
			}
			localPath := idDir + suffix
			if _, err := os.Stat(localPath); err == nil {
				TrailarrLog("Debug", "CacheMediaPosters", "Poster already exists: %s", localPath)
				continue
			}
			settings, err := loadMediaSettings(section)
			if err != nil {
				TrailarrLog("Warn", "CacheMediaPosters", "Failed to load settings for %s: %v", section, err)
				continue
			}
			apiBase := trimTrailingSlash(settings.URL)
			posterUrl := apiBase + RemoteMediaCoverPath + id + suffix
			TrailarrLog("Info", "CacheMediaPosters", "Attempting to cache poster for %s id=%s: %s -> %s", section, id, posterUrl, localPath)
			if err := fetchAndCachePoster(localPath, posterUrl, section); err != nil {
				TrailarrLog("Warn", "CacheMediaPosters", "Failed to cache poster for %s id=%s: %v", section, id, err)
			}
			TrailarrLog("Info", "CacheMediaPosters", "Successfully cached poster for %s id=%s: %s", section, id, localPath)
		}
	}
	TrailarrLog("Info", "CacheMediaPosters", "Finished poster caching for section: %s", section)
}

// Finds the media path for a given id in a cache file
func FindMediaPathByID(cachePath string, idStr string) (string, error) {
	items, err := loadCache(cachePath)
	if err != nil {
		return "", err
	}
	for _, m := range items {
		if mid, ok := m["id"]; ok && fmt.Sprintf("%v", mid) == idStr {
			if p, ok := m["path"].(string); ok {
				return p, nil
			}
			break
		}
	}
	return "", nil
}

// Scans a media path and returns a map of existing extras (type|title)
func ScanExistingExtras(mediaPath string) map[string]bool {
	existing := map[string]bool{}
	if mediaPath == "" {
		return existing
	}
	entries, err := os.ReadDir(mediaPath)
	if err != nil {
		return existing
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := mediaPath + "/" + entry.Name()
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mp4") {
				title := strings.TrimSuffix(f.Name(), ".mp4")
				key := entry.Name() + "|" + title
				existing[key] = true
			}
		}
	}
	return existing
}

// Checks which extras are downloaded in the given media path and marks them in the extras list
// extras: slice of map[string]string (from TMDB), mediaPath: path to the movie/series folder
// typeKey: the key in the extra map for the type (usually "type"), titleKey: the key for the title (usually "title")
func MarkDownloadedExtras(extras []map[string]string, mediaPath string, typeKey, titleKey string) {
	existing := ScanExistingExtras(mediaPath)
	for _, extra := range extras {
		// Canonicalize type and update the map so API always returns canonical type
		typeStr := canonicalizeExtraType(extra[typeKey], extra[titleKey])
		extra[typeKey] = typeStr
		title := SanitizeFilename(extra[titleKey])
		key := typeStr + "|" + title
		if existing[key] {
			extra["downloaded"] = "true"
		} else {
			extra["downloaded"] = "false"
		}
	}
}

// Common settings struct for both Radarr and Sonarr
// Use this for loading settings generically
type MediaSettings struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"apiKey"`
}

// Trims trailing slash from a URL
func trimTrailingSlash(url string) string {
	if strings.HasSuffix(url, "/") {
		return strings.TrimRight(url, "/")
	}
	return url
}

// Loads a JSON cache file into a generic slice
func loadCache(path string) ([]map[string]interface{}, error) {
	cacheData, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var items []map[string]interface{}
	if err := json.Unmarshal(cacheData, &items); err != nil {
		return nil, err
	}

	section, mainCachePath := detectSectionAndMainCachePath(path)
	titleMap := getTitleMap(mainCachePath, path)
	if section != "" {
		mappings, err := GetPathMappings(section)
		if err != nil {
			mappings = nil
		}
		for _, item := range items {
			updateItemPath(item, mappings)
			updateItemTitle(item, titleMap)
		}
	}

	return items, nil
}

// Helper: Detect section and main cache path
func detectSectionAndMainCachePath(path string) (string, string) {
	if strings.Contains(path, "movie") || strings.Contains(path, "Movie") {
		return "radarr", TrailarrRoot + "/movies.json"
	} else if strings.Contains(path, "series") || strings.Contains(path, "Series") {
		return "sonarr", TrailarrRoot + "/series.json"
	}
	return "", ""
}

// Helper: Get title map from main cache if needed
func getTitleMap(mainCachePath, path string) map[string]string {
	if mainCachePath == "" || mainCachePath == path {
		return nil
	}
	titleMap := make(map[string]string)
	mainCacheData, err := os.ReadFile(mainCachePath)
	if err != nil {
		return nil
	}
	var mainItems []map[string]interface{}
	if err := json.Unmarshal(mainCacheData, &mainItems); err != nil {
		return nil
	}
	for _, item := range mainItems {
		if id, ok := item["id"]; ok {
			if title, ok := item["title"].(string); ok {
				titleMap[fmt.Sprintf("%v", id)] = title
			}
		}
	}
	return titleMap
}

// Helper: Update item path using mappings
func updateItemPath(item map[string]interface{}, mappings [][]string) {
	p, ok := item["path"].(string)
	if !ok || p == "" || mappings == nil {
		return
	}
	for _, m := range mappings {
		if strings.HasPrefix(p, m[0]) {
			item["path"] = m[1] + p[len(m[0]):]
			break
		}
	}
}

// Helper: Update item title using title map
func updateItemTitle(item map[string]interface{}, titleMap map[string]string) {
	if titleMap != nil {
		if id, ok := item["id"]; ok {
			if title, exists := titleMap[fmt.Sprintf("%v", id)]; exists {
				item["title"] = title
			}
		}
	} else if title, ok := item["title"].(string); ok {
		item["title"] = title
	}
}

// Writes the wanted (no trailer) media to a JSON file
func writeWantedCache(section, cachePath, wantedPath string) error {
	items, err := loadCache(cachePath)
	if err != nil {
		return err
	}
	mappings, err := GetPathMappings(section)
	if err != nil {
		// If can't get mappings, use default paths
		mappings = nil
	}
	var trailerPaths []string
	for _, m := range mappings {
		if len(m) > 1 && m[1] != "" {
			trailerPaths = append(trailerPaths, m[1])
		}
	}
	if len(trailerPaths) == 0 {
		if section == "radarr" {
			trailerPaths = append(trailerPaths, "/Movies")
		} else {
			trailerPaths = append(trailerPaths, "/Series")
		}
	}
	trailerSet := findMediaWithTrailers(trailerPaths...)
	var wanted []map[string]interface{}
	for _, item := range items {
		path, ok := item["path"].(string)
		if !ok || trailerSet[path] {
			continue
		}
		wanted = append(wanted, item)
	}
	cacheData, _ := json.MarshalIndent(wanted, "", "  ")
	return os.WriteFile(wantedPath, cacheData, 0644)
}

// Move SyncMediaCacheJson to media.go for shared use
// Generic sync function for Radarr/Sonarr
// Syncs only the JSON cache for Radarr/Sonarr, not the media files themselves
// Pass section ("radarr" or "sonarr"), apiPath (e.g. "/api/v3/movie"), cachePath, and a filter function for items
func SyncMediaCacheJson(section, apiPath, cachePath string, filter func(map[string]interface{}) bool) error {
	url, apiKey, err := GetSectionUrlAndApiKey(section)
	if err != nil {
		TrailarrLog("Warn", "SyncMediaCacheJson", "%s settings not found: %v", section, err)
		return fmt.Errorf("%s settings not found: %w", section, err)
	}
	req, err := http.NewRequest("GET", url+apiPath, nil)
	if err != nil {
		TrailarrLog("Warn", "SyncMediaCacheJson", "error creating request: %v", err)
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		TrailarrLog("Warn", "SyncMediaCacheJson", "error fetching %s: %v", section, err)
		return fmt.Errorf("error fetching %s: %w", section, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		TrailarrLog("Warn", "SyncMediaCacheJson", "%s API error: %d", section, resp.StatusCode)
		return fmt.Errorf("%s API error: %d", section, resp.StatusCode)
	}
	var allItems []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allItems); err != nil {
		TrailarrLog("Warn", "SyncMediaCacheJson", "failed to decode %s response: %v", section, err)
		return fmt.Errorf("failed to decode %s response: %w", section, err)
	}
	items := make([]map[string]interface{}, 0)
	for _, m := range allItems {
		if filter(m) {
			items = append(items, m)
		}
	}
	cacheData, _ := json.MarshalIndent(items, "", "  ")
	_ = os.WriteFile(cachePath, cacheData, 0644)
	TrailarrLog("Info", "SyncMediaCacheJson", "[Sync%s] Synced %d items to cache.", section, len(items))

	// After syncing main cache, update wanted cache
	var wantedPath string
	if section == "radarr" {
		wantedPath = MoviesWantedFile
	} else {
		wantedPath = SeriesWantedFile
	}
	_ = writeWantedCache(section, cachePath, wantedPath)
	return nil
}

// Generic background sync for Radarr/Sonarr
func BackgroundSync(
	interval time.Duration,
	syncFunc func() error,
	queueAppend func(item interface{}),
	itemFactory func() interface{},
	itemUpdate func(item interface{}, started, ended time.Time, duration time.Duration, status, errStr string),
	queueTrim func(),
) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		item := itemFactory()
		queueAppend(item)
		started := time.Now()
		itemUpdate(item, started, started, 0, "running", "")
		err := syncFunc()
		ended := time.Now()
		duration := ended.Sub(started)
		status := "done"
		errStr := ""
		if err != nil {
			status = "error"
			errStr = err.Error()
		}
		itemUpdate(item, started, ended, duration, status, errStr)
		queueTrim()
		<-ticker.C
	}
}

// Returns a Gin handler to list media (movies/series) without any downloaded trailer extra
func GetMediaWithoutTrailerExtraHandler(section, cachePath, defaultPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use the wanted JSON file generated by background sync
		var wantedPath string
		if section == "radarr" {
			wantedPath = MoviesWantedFile
		} else {
			wantedPath = SeriesWantedFile
		}
		items, err := loadCache(wantedPath)
		if err != nil {
			c.JSON(500, gin.H{"error": section + " wanted cache not found"})
			return
		}
		if section == "radarr" {
			c.JSON(200, gin.H{"movies": items})
		} else {
			c.JSON(200, gin.H{"series": items})
		}
	}
}
