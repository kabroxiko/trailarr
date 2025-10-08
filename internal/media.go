package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

var GlobalSyncQueue []SyncQueueItem

// Generic function to sync media images for Radarr/Sonarr
func SyncMediaImages(provider, apiPath, cacheFile string, filter func(map[string]interface{}) bool, posterDir string, posterSuffixes []string) error {
	err := SyncMediaCacheJson(provider, apiPath, cacheFile, filter)
	if err != nil {
		return err
	}
	items, err := loadCache(cacheFile)
	if err != nil {
		return err
	}
	CacheMediaPosters(
		provider,
		posterDir,
		items,
		"id",
		posterSuffixes,
		true, // debug
	)
	return nil
}

// Generic handler for listing media (movies/series)
func GetMediaHandler(section, cacheFile, key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := loadCache(cacheFile)
		if err != nil {
			respondError(c, http.StatusInternalServerError, section+" cache not found")
			return
		}
		idParam := c.Query("id")
		filtered := items
		if idParam != "" {
			filtered = Filter(items, func(m map[string]interface{}) bool {
				id, ok := m[key]
				return ok && fmt.Sprintf("%v", id) == idParam
			})
		}
		respondJSON(c, http.StatusOK, gin.H{section: filtered})
	}
}

// parseMediaID parses an id from interface{} to int
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

// Parametric force sync for Radarr/Sonarr
type SyncQueueItem struct {
	TaskId   string
	Queued   time.Time
	Started  time.Time
	Ended    time.Time
	Duration time.Duration
	Status   string
	Error    string
}

// Parametric status struct for Radarr/Sonarr
type SyncStatus struct {
	LastExecution time.Time
	LastDuration  time.Duration
	NextExecution time.Time
	LastError     string
	Queue         []SyncQueueItem
}

func NewSyncStatus() *SyncStatus {
	return &SyncStatus{
		Queue: make([]SyncQueueItem, 0),
	}
}

// Parametric status getters
func LastExecution(status *SyncStatus) time.Time    { return status.LastExecution }
func LastDuration(status *SyncStatus) time.Duration { return status.LastDuration }
func NextExecution(status *SyncStatus) time.Time    { return status.NextExecution }
func LastError(status *SyncStatus) string           { return status.LastError }
func Queue(status *SyncStatus) []SyncQueueItem      { return status.Queue }

// UpdateSyncQueueItem updates status, timestamps, and error for a SyncQueueItem
func UpdateSyncQueueItem(item *SyncQueueItem, status string, started, ended time.Time, duration time.Duration, err error) {
	item.Status = status
	item.Started = started
	item.Ended = ended
	item.Duration = duration
	if err != nil {
		item.Error = err.Error()
	} else {
		item.Error = ""
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
	TrailarrLog(INFO, "SyncService", "Starting sync for section: %s", section)
	// Add new queue item to file on start
	queued := time.Now()
	item := SyncQueueItem{
		TaskId:  section,
		Queued:  queued,
		Status:  "running",
		Started: queued,
	}
	// Read queue file
	var fileQueue []SyncQueueItem
	_ = ReadJSONFile(QueueFile, &fileQueue)
	fileQueue = append(fileQueue, item)
	_ = WriteJSONFile(QueueFile, fileQueue)

	TrailarrLog(DEBUG, "SyncService", "Invoking syncFunc for section: %s", section)
	err := syncFunc()
	ended := time.Now()
	duration := ended.Sub(queued)
	status := "success"
	if err != nil {
		status = "failed"
		TrailarrLog(ERROR, "SyncService", "Sync %s error: %s", section, err.Error())
	} else {
		TrailarrLog(INFO, "SyncService", "Synced cache for %s.", section)
	}
	// Update the last queue item for this section (by TaskId and Queued)
	_ = ReadJSONFile(QueueFile, &fileQueue)
	for i := len(fileQueue) - 1; i >= 0; i-- {
		if fileQueue[i].TaskId == section && fileQueue[i].Queued.Equal(queued) {
			fileQueue[i].Status = status
			fileQueue[i].Started = queued
			fileQueue[i].Ended = ended
			fileQueue[i].Duration = duration
			if err != nil {
				fileQueue[i].Error = err.Error()
			} else {
				fileQueue[i].Error = ""
			}
			break
		}
	}
	_ = WriteJSONFile(QueueFile, fileQueue)
	TrailarrLog(INFO, "SyncService", "Finished sync for section: %s", section)
	*lastExecution = ended
	*lastDuration = duration
	interval := timings[section]
	*nextExecution = ended.Add(time.Duration(interval) * time.Minute)
}

// Helper to fetch and cache poster image
func fetchAndCachePoster(localPath, posterUrl, section string) error {
	resp, err := http.Get(posterUrl)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		CheckErrLog(WARN, "CacheMediaPosters", "Failed to fetch poster image", err)
		return fmt.Errorf("failed to fetch poster image from %s", section)
	}
	defer resp.Body.Close()
	out, err := os.Create(localPath)
	if CheckErrLog(WARN, "CacheMediaPosters", "Failed to cache poster image", err) != nil {
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
	TrailarrLog(INFO, "CacheMediaPosters", "Starting poster caching for section: %s, baseDir: %s, items: %d", section, baseDir, len(idList))
	type posterJob struct {
		id        string
		idDir     string
		localPath string
		posterUrl string
	}
	for _, item := range idList {
		id := fmt.Sprintf("%v", item[idKey])
		settings, err := loadMediaSettings(section)
		if CheckErrLog(WARN, "CacheMediaPosters", "Failed to load settings", err) != nil {
			continue
		}
		apiBase := trimTrailingSlash(settings.URL)
		jobs := Map(posterSuffixes, func(suffix string) posterJob {
			idDir := baseDir + "/" + id
			localPath := idDir + suffix
			posterUrl := apiBase + RemoteMediaCoverPath + id + suffix
			return posterJob{id, idDir, localPath, posterUrl}
		})
		for _, job := range jobs {
			if err := os.MkdirAll(job.idDir, 0775); CheckErrLog(WARN, "CacheMediaPosters", "Failed to create dir", err) != nil {
				continue
			}
			if _, err := os.Stat(job.localPath); err == nil {
				continue
			}
			TrailarrLog(INFO, "CacheMediaPosters", "Attempting to cache poster for %s id=%s: %s -> %s", section, job.id, job.posterUrl, job.localPath)
			if err := fetchAndCachePoster(job.localPath, job.posterUrl, section); err != nil {
				TrailarrLog(WARN, "CacheMediaPosters", "Failed to cache poster for %s id=%s: %v", section, job.id, err)
			}
			TrailarrLog(INFO, "CacheMediaPosters", "Successfully cached poster for %s id=%s: %s", section, job.id, job.localPath)
		}
	}
	TrailarrLog(INFO, "CacheMediaPosters", "Finished poster caching for section: %s", section)
}

// Finds the media path for a given id in a cache file
func FindMediaPathByID(cacheFile string, mediaId int) (string, error) {
	items, err := loadCache(cacheFile)
	if CheckErrLog(WARN, "FindMediaPathByID", "Failed to load cache", err) != nil {
		return "", err
	}
	for _, item := range items {
		idInt, ok := parseMediaID(item["id"])
		if !ok {
			continue
		}
		if idInt == mediaId {
			if p, ok := item["path"].(string); ok {
				return p, nil
			}
			break
		}
	}
	return "", nil
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
	var items []map[string]interface{}
	if err := ReadJSONFile(path, &items); CheckErrLog(WARN, "loadCache", "ReadJSONFile failed", err) != nil {
		return nil, err
	}

	mediaType, mainCachePath := detectMediaTypeAndMainCachePath(path)
	titleMap := getTitleMap(mainCachePath, path)
	if mediaType != "" {
		mappings, err := GetPathMappings(mediaType)
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

// Helper: Detect media type and main cache path
func detectMediaTypeAndMainCachePath(path string) (MediaType, string) {
	if strings.Contains(path, "movie") || strings.Contains(path, "Movie") {
		return MediaTypeMovie, MoviesJSONPath
	} else if strings.Contains(path, "series") || strings.Contains(path, "Series") {
		return MediaTypeTV, SeriesJSONPath
	}
	return "", ""
}

// Helper: Get title map from main cache if needed
func getTitleMap(mainCachePath, path string) map[string]string {
	if mainCachePath == "" || mainCachePath == path {
		return nil
	}
	titleMap := make(map[string]string)
	var mainItems []map[string]interface{}
	if err := ReadJSONFile(mainCachePath, &mainItems); CheckErrLog(WARN, "getTitleMap", "ReadJSONFile failed", err) != nil {
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
func writeWantedCache(mediaType MediaType, cacheFile, wantedPath string) error {
	items, err := loadCache(cacheFile)
	if CheckErrLog(WARN, "writeWantedCache", "Failed to load cache", err) != nil {
		return err
	}
	mappings, err := GetPathMappings(mediaType)
	if CheckErrLog(WARN, "writeWantedCache", "GetPathMappings failed", err) != nil {
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
		if mediaType == MediaTypeMovie {
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
	return WriteJSONFile(wantedPath, wanted)
}

// Generic sync function for Radarr/Sonarr
// Syncs only the JSON cache for Radarr/Sonarr, not the media files themselves
// Pass mediaType (MediaTypeMovie or MediaTypeTV), apiPath (e.g. "/api/v3/movie"), cacheFile, and a filter function for items
func SyncMediaCacheJson(provider, apiPath, cacheFile string, filter func(map[string]interface{}) bool) error {
	url, apiKey, err := GetProviderUrlAndApiKey(provider)
	if CheckErrLog(WARN, "SyncMediaCacheJson", "settings not found", err) != nil {
		return fmt.Errorf("%s settings not found: %w", provider, err)
	}
	req, err := http.NewRequest("GET", url+apiPath, nil)
	if CheckErrLog(WARN, "SyncMediaCacheJson", "error creating request", err) != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if CheckErrLog(WARN, "SyncMediaCacheJson", "error fetching", err) != nil {
		return fmt.Errorf("error fetching %s: %w", provider, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		TrailarrLog(WARN, "SyncMediaCacheJson", "%s API error: %d", provider, resp.StatusCode)
		return fmt.Errorf("%s API error: %d", provider, resp.StatusCode)
	}
	var allItems []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allItems); CheckErrLog(WARN, "SyncMediaCacheJson", "failed to decode response", err) != nil {
		return fmt.Errorf("failed to decode %s response: %w", provider, err)
	}
	items := make([]map[string]interface{}, 0)
	for _, m := range allItems {
		if filter(m) {
			items = append(items, m)
		}
	}
	_ = WriteJSONFile(cacheFile, items)
	TrailarrLog(INFO, "SyncMediaCacheJson", "[Sync%s] Synced %d items to cache.", provider, len(items))

	// After syncing main cache, update wanted cache
	var wantedPath string
	var mediaType MediaType
	if provider == "radarr" {
		wantedPath = MoviesWantedFile
		mediaType = MediaTypeMovie
	} else {
		wantedPath = SeriesWantedFile
		mediaType = MediaTypeTV
	}
	_ = writeWantedCache(mediaType, cacheFile, wantedPath)
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
func GetMediaWithoutTrailerExtraHandler(section, cacheFile, defaultPath string) gin.HandlerFunc {
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
			respondError(c, http.StatusInternalServerError, section+" wanted cache not found")
			return
		}
		if section == "radarr" {
			respondJSON(c, http.StatusOK, gin.H{"movies": items})
		} else {
			respondJSON(c, http.StatusOK, gin.H{"series": items})
		}
	}
}

// sharedExtrasHandler handles extras for both movies and series
func sharedExtrasHandler(mediaType MediaType) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		var id int
		fmt.Sscanf(idStr, "%d", &id)
		extras, err := SearchExtras(mediaType, id)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		cacheFile, _ := resolveCachePath(mediaType)
		mediaPath, err := FindMediaPathByID(cacheFile, id)
		if err != nil {
			respondError(c, http.StatusInternalServerError, fmt.Sprintf("%s cache not found", mediaType))
			return
		}
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		rejectedExtras := GetRejectedExtrasForMedia(mediaType, id)
		TrailarrLog(DEBUG, "sharedExtrasHandler", "Rejected extras: %+v", rejectedExtras)
		youtubeIdInResults := make(map[string]struct{})
		for _, extra := range extras {
			youtubeIdInResults[extra.YoutubeId] = struct{}{}
		}
		// Set status to "rejected" for any extra whose URL matches a rejected extra
		rejectedYoutubeIds := make(map[string]RejectedExtra)
		for _, rejected := range rejectedExtras {
			rejectedYoutubeIds[rejected.YoutubeId] = rejected
		}
		for i, extra := range extras {
			if _, exists := rejectedYoutubeIds[extra.YoutubeId]; exists {
				extras[i].Status = "rejected"
			}
		}
		// Also append any rejected extras not already present in extras
		for _, rejected := range rejectedExtras {
			if _, exists := youtubeIdInResults[rejected.YoutubeId]; !exists {
				extras = append(extras, Extra{
					Type:      rejected.ExtraType,
					Title:     rejected.ExtraTitle,
					YoutubeId: rejected.YoutubeId,
					Status:    "rejected",
				})
			}
		}
		TrailarrLog(DEBUG, "sharedExtrasHandler", "Extras response: %+v", extras)
		respondJSON(c, http.StatusOK, gin.H{"extras": extras})
	}
}

// respondError is a helper for Gin error responses
func respondError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

// respondJSON is a helper for Gin JSON responses
func respondJSON(c *gin.Context, code int, obj interface{}) {
	c.JSON(code, obj)
}
