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

// Generic handler for Radarr/Sonarr sync status
func GetSyncStatusHandler(section string, status *SyncStatus, displayName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		interval := Timings[section]
		respondJSON(c, http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          "Sync with " + displayName,
				"interval":      fmt.Sprintf("%d minutes", interval),
				"lastExecution": LastExecution(status),
				"lastDuration":  LastDuration(status).String(),
				"nextExecution": NextExecution(status),
				"lastError":     LastError(status),
			},
			"queue": Queue(status),
		})
	}
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

var GlobalSyncQueue []SyncQueueItem

const (
	MoviesWantedFile = TrailarrRoot + "/movies_wanted.json"
	SeriesWantedFile = TrailarrRoot + "/series_wanted.json"
	queueFile        = TrailarrRoot + "/queue.json"
)

// DownloadMissingExtras downloads missing extras for a given media type ("movie" or "tv")
func DownloadMissingExtras(mediaType MediaType, cacheFile string) error {
	TrailarrLog(INFO, "DownloadMissingExtras", "DownloadMissingExtras: mediaType=%s, cacheFile=%s", mediaType, cacheFile)
	items, err := loadCache(cacheFile)
	if CheckErrLog(WARN, "DownloadMissingExtras", "Failed to load cache", err) != nil {
		return err
	}
	type downloadItem struct {
		idInt     int
		mediaPath string
		extras    []Extra
	}
	config, _ := GetExtraTypesConfig()
	filtered := Filter(items, func(m map[string]interface{}) bool {
		idInt, ok := parseMediaID(m["id"])
		if !ok {
			TrailarrLog(WARN, "DownloadMissingExtras", "Missing or invalid id in item: %v", m)
			return false
		}
		_, err := SearchExtras(mediaType, idInt)
		if err != nil {
			TrailarrLog(WARN, "DownloadMissingExtras", "SearchExtras error: %v", err)
			return false
		}
		mediaPath, err := FindMediaPathByID(cacheFile, idInt)
		if err != nil || mediaPath == "" {
			TrailarrLog(WARN, "DownloadMissingExtras", "FindMediaPathByID error or empty: %v, mediaPath=%s", err, mediaPath)
			return false
		}
		return true
	})
	mapped := Map(filtered, func(media map[string]interface{}) downloadItem {
		idInt, _ := parseMediaID(media["id"])
		extras, _ := SearchExtras(mediaType, idInt)
		mediaPath, _ := FindMediaPathByID(cacheFile, idInt)
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		// Defensive: mark rejected extras before any download
		rejectedExtras := GetRejectedExtrasForMedia(mediaType, idInt)
		rejectedYoutubeIds := make(map[string]struct{})
		for _, r := range rejectedExtras {
			rejectedYoutubeIds[r.YoutubeId] = struct{}{}
		}
		for i := range extras {
			if _, exists := rejectedYoutubeIds[extras[i].YoutubeId]; exists {
				extras[i].Status = "rejected"
			}
		}
		return downloadItem{idInt, mediaPath, extras}
	})
	for _, di := range mapped {
		filterAndDownloadExtras(mediaType, di.idInt, di.extras, config)
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

func filterAndDownloadExtras(mediaType MediaType, mediaId int, extras []Extra, config ExtraTypesConfig) {
	// Mark extras as rejected if their YouTube ID matches any in rejected_extras.json
	rejectedExtras := GetRejectedExtrasForMedia(mediaType, mediaId)
	rejectedYoutubeIds := make(map[string]struct{})
	for _, r := range rejectedExtras {
		rejectedYoutubeIds[r.YoutubeId] = struct{}{}
	}
	for i := range extras {
		if _, exists := rejectedYoutubeIds[extras[i].YoutubeId]; exists {
			extras[i].Status = "rejected"
		}
	}
	filtered := Filter(extras, func(extra Extra) bool {
		return shouldDownloadExtra(extra, config)
	})
	for _, extra := range filtered {
		err := handleExtraDownload(mediaType, mediaId, extra)
		CheckErrLog(WARN, "DownloadMissingExtras", "Failed to download", err)
	}
}

func shouldDownloadExtra(extra Extra, config ExtraTypesConfig) bool {
	if extra.Status != "missing" || extra.YoutubeId == "" {
		return false
	}
	if extra.Status == "rejected" {
		return false
	}
	typeName := extra.Type
	canonical := canonicalizeExtraType(typeName, "")
	return isExtraTypeEnabled(config, canonical)
}

func handleExtraDownload(mediaType MediaType, mediaId int, extra Extra) error {
	if extra.Status == "rejected" {
		TrailarrLog(INFO, "DownloadMissingExtras", "Skipping rejected extra: mediaType=%v, mediaId=%v, type=%s, title=%s, youtubeId=%s", mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
		return nil
	}
	_, err := DownloadYouTubeExtra(mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
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
	TrailarrLog(INFO, "SyncService", "Starting sync for section: %s", section)
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

	started := time.Now()
	UpdateSyncQueueItem(&GlobalSyncQueue[idx], "running", started, started, 0, nil)
	saveQueue()

	TrailarrLog(DEBUG, "SyncService", "Invoking syncFunc for section: %s", section)
	err := syncFunc()
	ended := time.Now()
	duration := ended.Sub(started)
	status := "success"
	if err != nil {
		status = "failed"
		TrailarrLog(ERROR, "SyncService", "Sync %s error: %s", section, err.Error())
	} else {
		TrailarrLog(INFO, "SyncService", "Synced cache for %s.", section)
	}
	UpdateSyncQueueItem(&GlobalSyncQueue[idx], status, started, ended, duration, err)
	saveQueue()
	TrailarrLog(INFO, "SyncService", "Finished sync for section: %s", section)
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

// Scans a media path and returns a map of existing extras (type|title)
func ScanExistingExtras(mediaPath string) map[string]bool {
	existing := map[string]bool{}
	if mediaPath == "" {
		return existing
	}
	entries, err := os.ReadDir(mediaPath)
	if CheckErrLog(WARN, "ScanExistingExtras", "ReadDir failed", err) != nil {
		return existing
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := mediaPath + "/" + entry.Name()
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mkv") {
				title := strings.TrimSuffix(f.Name(), ".mkv")
				key := entry.Name() + "|" + title
				existing[key] = true
			}
		}
	}
	return existing
}

// Checks which extras are downloaded in the given media path and marks them in the extras list
// extras: slice of Extra (from TMDB), mediaPath: path to the movie/series folder
// typeKey: the key in the extra map for the type (usually "type"), titleKey: the key for the title (usually "title")
func MarkDownloadedExtras(extras []Extra, mediaPath string, typeKey, titleKey string) {
	existing := ScanExistingExtras(mediaPath)
	for i := range extras {
		typeStr := canonicalizeExtraType(extras[i].Type, extras[i].Title)
		extras[i].Type = typeStr
		title := SanitizeFilename(extras[i].Title)
		key := typeStr + "|" + title
		extras[i].Status = "missing"
		if existing[key] {
			extras[i].Status = "downloaded"
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
		return MediaTypeMovie, TrailarrRoot + "/movies.json"
	} else if strings.Contains(path, "series") || strings.Contains(path, "Series") {
		return MediaTypeTV, TrailarrRoot + "/series.json"
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
