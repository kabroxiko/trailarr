package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

const (
	cacheControlHeader = "Cache-Control"
	cacheControlValue  = "public, max-age=86400"
	totalTimeLogFormat = "Total time: %v"
)

// SearchExtras merges extras from the main cache and the persistent extras collection for a media item
func SearchExtras(mediaType MediaType, mediaId int) ([]Extra, error) {
	ctx := context.Background()
	entries, err := GetExtrasForMedia(ctx, mediaType, mediaId)
	if err != nil {
		// fallback to old method if needed
		persistent, _ := GetAllExtras(ctx)
		entries = make([]ExtrasEntry, 0)
		for _, e := range persistent {
			if e.MediaType == mediaType && e.MediaId == mediaId {
				entries = append(entries, e)
			}
		}
	}
	result := make([]Extra, 0, len(entries))
	for _, e := range entries {
		result = append(result, Extra{
			ExtraType:  e.ExtraType,
			ExtraTitle: e.ExtraTitle,
			YoutubeId:  e.YoutubeId,
			Status:     e.Status,
		})
	}
	return result, nil
}

// ProxyYouTubeImageHandler proxies YouTube thumbnail images to avoid 404s and CORS issues
func ProxyYouTubeImageHandler(c *gin.Context) {
	youtubeId := c.Param("youtubeId")
	if youtubeId == "" {
		respondError(c, http.StatusBadRequest, "Missing youtubeId")
		return
	}

	cacheDir := filepath.Join(MediaCoverPath, "YouTube")
	ensureDirIfNeeded(MediaCoverPath, "MediaCoverPath")
	ensureDirIfNeeded(cacheDir, "cacheDir")

	if path, ct := cachedYouTubeImage(cacheDir, youtubeId); path != "" {
		serveCachedFile(c, path, ct)
		return
	}

	thumbUrls := []string{
		"https://i.ytimg.com/vi/" + youtubeId + "/maxresdefault.jpg",
		"https://i.ytimg.com/vi/" + youtubeId + "/hqdefault.jpg",
	}
	resp, err := fetchFirstSuccessful(thumbUrls)
	if err != nil || resp == nil {
		serveFallbackSVG(c)
		return
	}
	defer resp.Body.Close()

	ct := resp.Header.Get(HeaderContentType)
	ext := detectImageExt(ct)
	tmpPath := filepath.Join(cacheDir, youtubeId+".tmp")
	finalPath := filepath.Join(cacheDir, youtubeId+ext)

	if err := saveToTmp(resp.Body, tmpPath); err != nil {
		// couldn't cache; stream the response directly
		streamResponse(c, ct, resp.Body)
		return
	}
	_ = os.Rename(tmpPath, finalPath)
	serveCachedFile(c, finalPath, ct)
}

// helper: ensure directory exists but don't fail the whole handler
func ensureDirIfNeeded(path, context string) {
	if err := os.MkdirAll(path, 0775); err != nil {
		TrailarrLog(WARN, "ProxyYouTubeImageHandler", "Failed to create %s %s: %v", context, path, err)
	}
}

// helper: check cached files and return path + content type
func cachedYouTubeImage(cacheDir, id string) (string, string) {
	exts := []struct {
		ext string
		ct  string
	}{
		{".jpg", "image/jpeg"},
		{".jpeg", "image/jpeg"},
		{".png", "image/png"},
		{".webp", "image/webp"},
		{".svg", "image/svg+xml"},
	}
	for _, e := range exts {
		p := filepath.Join(cacheDir, id+e.ext)
		if _, err := os.Stat(p); err == nil {
			return p, e.ct
		}
	}
	return "", ""
}

func serveCachedFile(c *gin.Context, path, contentType string) {
	c.Header(HeaderContentType, contentType)
	c.Header(cacheControlHeader, cacheControlValue)
	if c.Request.Method == http.MethodHead {
		c.Status(http.StatusOK)
		return
	}
	c.File(path)
}

// helper: fetch the first successful response from candidate URLs
func fetchFirstSuccessful(urls []string) (*http.Response, error) {
	for _, u := range urls {
		resp, err := http.Get(u)
		if err != nil {
			if resp != nil {
				resp.Body.Close()
			}
			continue
		}
		if resp.StatusCode == 200 {
			return resp, nil
		}
		resp.Body.Close()
	}
	return nil, fmt.Errorf("no successful response")
}

func serveFallbackSVG(c *gin.Context) {
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="64" height="64" role="img" aria-label="Unavailable">
  <circle cx="64" cy="64" r="30" fill="none" stroke="#888" stroke-width="8" />
  <!-- diagonal from top-right to bottom-left -->
  <line x1="92" y1="36" x2="36" y2="92" stroke="#888" stroke-width="10" stroke-linecap="round" />
</svg>`
	c.Header(HeaderContentType, "image/svg+xml")
	c.Header("X-Proxy-Fallback", "1")
	c.Header(cacheControlHeader, cacheControlValue)
	c.Status(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return
	}
	_, _ = c.Writer.Write([]byte(svg))
}

// helper: determine extension from content type
func detectImageExt(ct string) string {
	switch {
	case strings.Contains(ct, "jpeg"):
		return ".jpg"
	case strings.Contains(ct, "png"):
		return ".png"
	case strings.Contains(ct, "webp"):
		return ".webp"
	case strings.Contains(ct, "svg"):
		return ".svg"
	default:
		return ".jpg"
	}
}

// helper: save response body to tmp file
func saveToTmp(r io.Reader, path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, r)
	return err
}

func streamResponse(c *gin.Context, ct string, r io.Reader) {
	c.Header(HeaderContentType, ct)
	c.Header(cacheControlHeader, cacheControlValue)
	c.Status(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return
	}
	_, _ = io.Copy(c.Writer, r)
}

// Syncs media cache and caches poster images for Radarr/Sonarr
func SyncMedia(provider, apiPath, cacheFile string, filter func(map[string]interface{}) bool, posterDir string, posterSuffixes []string) error {
	// Minimal fast sync: fetch the list from provider, apply filter, save to cache.
	// Skip extras scanning, poster caching and new-item background processing to keep this fast.
	start := time.Now()
	TrailarrLog(DEBUG, "SyncMedia", "Starting fast SyncMedia (minimal): provider=%s apiPath=%s cacheFile=%s", provider, apiPath, cacheFile)

	allItems, err := fetchProviderItems(provider, apiPath)
	if err != nil {
		TrailarrLog(WARN, "SyncMedia", "Failed to fetch items from provider=%s apiPath=%s: %v", provider, apiPath, err)
		return err
	}

	filtered := make([]map[string]interface{}, 0, len(allItems))
	for _, m := range allItems {
		if filter == nil || filter(m) {
			filtered = append(filtered, m)
		}
	}

	if err := saveItems(cacheFile, filtered); err != nil {
		TrailarrLog(WARN, "SyncMedia", "Failed to save cache %s: %v", cacheFile, err)
		return err
	}
	// Cache poster images for the filtered items as part of sync (best-effort).
	// Run poster caching synchronously here so the cache is populated immediately.
	CacheMediaPosters(
		provider,
		posterDir,
		filtered,
		"id",
		posterSuffixes,
		true, // debug
	)

	TrailarrLog(INFO, "SyncMedia", "Fast SyncMedia completed: provider=%s saved=%d duration=%v", provider, len(filtered), time.Since(start))
	return nil
}

// Generic handler for listing media (movies/series)
func GetMediaHandler(cacheFile, key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := loadCache(cacheFile)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "cache not found")
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
		respondJSON(c, http.StatusOK, gin.H{"items": filtered})
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

// Helper to fetch and cache poster image
func fetchAndCachePoster(localPath, posterUrl, section string) error {
	resp, err := http.Get(posterUrl)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		TrailarrLog(WARN, "CacheMediaPosters", "Failed to fetch poster image: %v", err)
		return fmt.Errorf("failed to fetch poster image from %s", section)
	}
	defer resp.Body.Close()
	out, err := os.Create(localPath)
	if err != nil {
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

	// Load provider settings once
	settings, err := loadMediaSettings(section)
	if err != nil {
		TrailarrLog(WARN, "CacheMediaPosters", "Failed to load media settings for section=%s: %v", section, err)
		return
	}
	apiBase := trimTrailingSlash(settings.ProviderURL)

	// Build jobs
	jobsList := make([]posterJob, 0, len(idList)*len(posterSuffixes))
	for _, item := range idList {
		id := fmt.Sprintf("%v", item[idKey])
		idDir := baseDir + "/" + id
		for _, suffix := range posterSuffixes {
			localPath := idDir + suffix
			posterUrl := apiBase + RemoteMediaCoverPath + id + suffix
			jobsList = append(jobsList, posterJob{id, idDir, localPath, posterUrl})
		}
	}

	if len(jobsList) == 0 {
		TrailarrLog(DEBUG, "CacheMediaPosters", "No poster jobs to process for section=%s", section)
		return
	}

	// Worker pool
	maxWorkers := 8
	if len(jobsList) < maxWorkers {
		maxWorkers = len(jobsList)
	}
	jobs := make(chan posterJob, len(jobsList))
	var wg sync.WaitGroup
	var success int64
	var failed int64

	for w := 0; w < maxWorkers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for job := range jobs {
				// ensure directory exists
				if err := os.MkdirAll(job.idDir, 0775); err != nil {
					atomic.AddInt64(&failed, 1)
					TrailarrLog(DEBUG, "CacheMediaPosters", "worker=%d failed to create dir for id=%s: %v", workerID, job.id, err)
					continue
				}
				// skip if already cached
				if _, err := os.Stat(job.localPath); err == nil {
					atomic.AddInt64(&success, 1)
					continue
				}
				TrailarrLog(DEBUG, "CacheMediaPosters", "worker=%d downloading poster for id=%s: %s -> %s", workerID, job.id, job.posterUrl, job.localPath)
				if err := fetchAndCachePoster(job.localPath, job.posterUrl, section); err != nil {
					atomic.AddInt64(&failed, 1)
					TrailarrLog(WARN, "CacheMediaPosters", "worker=%d failed to cache poster for %s id=%s: %v", workerID, section, job.id, err)
					continue
				}
				atomic.AddInt64(&success, 1)
				TrailarrLog(DEBUG, "CacheMediaPosters", "worker=%d successfully cached poster for id=%s: %s", workerID, job.id, job.localPath)
			}
		}(w)
	}

	// Enqueue jobs
	for _, j := range jobsList {
		jobs <- j
	}
	close(jobs)
	wg.Wait()

	TrailarrLog(INFO, "CacheMediaPosters", "Finished poster caching for section=%s workers=%d jobs=%d success=%d failed=%d", section, maxWorkers, len(jobsList), atomic.LoadInt64(&success), atomic.LoadInt64(&failed))
}

// Finds the media path for a given id in a cache file
func FindMediaPathByID(cacheFile string, mediaId int) (string, error) {
	items, err := loadCache(cacheFile)
	if err != nil {
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
	ProviderURL string `yaml:"url"`
	APIKey      string `yaml:"apiKey"`
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
	// Use store for movies and series
	if path == MoviesStoreKey || path == SeriesStoreKey {
		items, err := LoadMediaFromStore(path)
		if err != nil {
			return nil, err
		}
		return processLoadedItems(items, path), nil
	}

	// Fallback to file for other paths
	var items []map[string]interface{}
	if err := ReadJSONFile(path, &items); err != nil {
		return nil, err
	}
	return processLoadedItems(items, path), nil
}

// Helper to apply path mappings and title map to loaded items, if applicable.
func processLoadedItems(items []map[string]interface{}, path string) []map[string]interface{} {
	mediaType, mainCachePath := detectMediaTypeAndMainCachePath(path)
	if mediaType == "" {
		return items
	}
	titleMap := getTitleMap(mainCachePath, path)
	mappings, err := GetPathMappings(mediaType)
	if err != nil {
		mappings = nil
	}
	for _, item := range items {
		updateItemPath(item, mappings)
		updateItemTitle(item, titleMap)
		// Do NOT attach extras from collection; extras are only in the extras collection now
	}
	return items
}

// LoadMediaFromStore loads movies or series from the persistent store.
// Expects path to be MoviesStoreKey or SeriesStoreKey.
func LoadMediaFromStore(path string) ([]map[string]interface{}, error) {
	client := GetStoreClient()
	ctx := context.Background()
	var storeKey string
	switch path {
	case MoviesStoreKey:
		storeKey = "trailarr:movies"
	case SeriesStoreKey:
		storeKey = "trailarr:series"
	default:
		return nil, fmt.Errorf("unsupported path for bbolt: %s", path)
	}
	val, err := client.Get(ctx, storeKey)
	if err != nil {
		if err == ErrNotFound {
			return []map[string]interface{}{}, nil // treat as empty
		}
		return nil, err
	}
	var items []map[string]interface{}
	if err := json.Unmarshal([]byte(val), &items); err != nil {
		return nil, err
	}
	return items, nil
}

// SaveMediaToStore saves movies or series to the persistent store.
// Expects path to be MoviesStoreKey or SeriesStoreKey.
func SaveMediaToStore(path string, items []map[string]interface{}) error {
	client := GetStoreClient()
	ctx := context.Background()
	var storeKey string
	switch path {
	case MoviesStoreKey:
		storeKey = "trailarr:movies"
	case SeriesStoreKey:
		storeKey = "trailarr:series"
	default:
		return fmt.Errorf("unsupported path for bbolt: %s", path)
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return client.Set(ctx, storeKey, data)
}

// Helper: Detect media type and main cache path
func detectMediaTypeAndMainCachePath(path string) (MediaType, string) {
	if strings.Contains(path, "movie") || strings.Contains(path, "Movie") {
		return MediaTypeMovie, MoviesStoreKey
	} else if strings.Contains(path, "series") || strings.Contains(path, "Series") {
		return MediaTypeTV, SeriesStoreKey
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
	if err := ReadJSONFile(mainCachePath, &mainItems); err != nil {
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

// (Removed: see updateWantedStatusInMainJson)

// Generic sync function for Radarr/Sonarr
// Syncs only the JSON cache for Radarr/Sonarr, not the media files themselves
// Pass mediaType (MediaTypeMovie or MediaTypeTV), apiPath (e.g. "/api/v3/movie"), cacheFile, and a filter function for items
func SyncMediaCache(provider, apiPath, cacheFile string, filter func(map[string]interface{}) bool) error {
	start := time.Now()
	TrailarrLog(DEBUG, "SyncMediaCache", "Starting SyncMediaCache: provider=%s apiPath=%s cacheFile=%s", provider, apiPath, cacheFile)

	allItems, err := fetchProviderItems(provider, apiPath)
	if err != nil {
		TrailarrLog(WARN, "SyncMediaCache", "Failed to fetch items from provider=%s apiPath=%s: %v", provider, apiPath, err)
		return err
	}
	TrailarrLog(DEBUG, "SyncMediaCache", "Fetched %d items from provider %s", len(allItems), provider)

	ctx := context.Background()
	items := collectFilteredItems(ctx, provider, allItems, filter)
	TrailarrLog(DEBUG, "SyncMediaCache", "Filtered items count for provider %s: %d", provider, len(items))

	// Load previous items (if any) so we can detect newly added media
	prevItems, _ := loadCache(cacheFile)
	TrailarrLog(DEBUG, "SyncMediaCache", "Previous cache size for %s: %d", cacheFile, len(prevItems))

	// Save items to the appropriate backend
	if err := saveItems(cacheFile, items); err != nil {
		TrailarrLog(WARN, "SyncMediaCache", "Failed to save items to %s: %v", cacheFile, err)
	} else {
		TrailarrLog(DEBUG, "SyncMediaCache", "Saved %d items to %s", len(items), cacheFile)
	}

	// Handle new items (best-effort, background tasks)
	handleNewItems(provider, items, prevItems)
	TrailarrLog(DEBUG, "SyncMediaCache", "Triggered background processing for new items (provider=%s)", provider)

	TrailarrLog(INFO, "SyncMediaCache", "[Sync%s] Synced %d items to cache. duration=%v", provider, len(items), time.Since(start))

	// After syncing main cache, update wanted status in main JSON
	var mediaType MediaType
	if provider == "radarr" {
		mediaType = MediaTypeMovie
	} else {
		mediaType = MediaTypeTV
	}
	if err := updateWantedStatusInMainJson(mediaType, cacheFile); err != nil {
		TrailarrLog(WARN, "SyncMediaCache", "updateWantedStatusInMainJson failed for %s: %v", cacheFile, err)
	} else {
		TrailarrLog(DEBUG, "SyncMediaCache", "updateWantedStatusInMainJson completed for %s", cacheFile)
	}
	return nil
}

// collectFilteredItems applies the filter and records extras for each accepted item.
func collectFilteredItems(ctx context.Context, provider string, allItems []map[string]interface{}, filter func(map[string]interface{}) bool) []map[string]interface{} {
	TrailarrLog(DEBUG, "collectFilteredItems", "Starting collectFilteredItems for provider=%s total_items=%d", provider, len(allItems))
	start := time.Now()
	total := len(allItems)

	// First: filter items (no extras ingestion yet) so we can dispatch ingestion
	items := make([]map[string]interface{}, 0)
	for idx, m := range allItems {
		TrailarrLog(DEBUG, "collectFilteredItems", "Inspecting item %d/%d for provider=%s id=%v", idx+1, total, provider, m["id"])
		if !filter(m) {
			continue
		}
		items = append(items, m)
	}

	filtered := len(items)
	if filtered == 0 {
		TrailarrLog(DEBUG, "collectFilteredItems", "provider=%s total_fetched=%d filtered=0 duration=%v", provider, total, time.Since(start))
		return items
	}

	// Dispatch extras ingestion asynchronously using a bounded worker pool.
	// We buffer the jobs channel to avoid blocking; workers run in background and process jobs.
	jobs := make(chan map[string]interface{}, filtered)
	workerCount := 8
	if filtered < workerCount {
		workerCount = filtered
	}

	for w := 0; w < workerCount; w++ {
		go func(workerID int) {
			for m := range jobs {
				if err := addExtrasFromItem(ctx, provider, m); err != nil {
					TrailarrLog(DEBUG, "collectFilteredItems", "async addExtrasFromItem error provider=%s worker=%d id=%v: %v", provider, workerID, m["id"], err)
				}
			}
		}(w)
	}

	// enqueue jobs (non-blocking because channel is buffered to filtered)
	for _, m := range items {
		jobs <- m
	}
	close(jobs)

	TrailarrLog(DEBUG, "collectFilteredItems", "provider=%s total_fetched=%d filtered=%d dispatched_extras_jobs=%d workers=%d duration=%v", provider, total, filtered, filtered, workerCount, time.Since(start))
	return items
}

// saveItems persists items either to the embedded store or to a file depending on cacheFile.
func saveItems(cacheFile string, items []map[string]interface{}) error {
	if cacheFile == MoviesStoreKey || cacheFile == SeriesStoreKey {
		return SaveMediaToStore(cacheFile, items)
	}
	return WriteJSONFile(cacheFile, items)
}

// handleNewItems detects newly added items and triggers background processing for each.
func handleNewItems(provider string, items, prevItems []map[string]interface{}) {
	if len(prevItems) == 0 {
		return
	}
	prevIDs := make(map[int]struct{}, len(prevItems))
	for _, pi := range prevItems {
		if idRaw, ok := pi["id"]; ok {
			if idInt, ok2 := parseMediaID(idRaw); ok2 {
				prevIDs[idInt] = struct{}{}
			}
		}
	}

	mediaType := MediaTypeMovie
	if provider == "sonarr" {
		mediaType = MediaTypeTV
	}

	cfg, _ := GetExtraTypesConfig()

	for _, it := range items {
		idRaw, ok := it["id"]
		if !ok {
			continue
		}
		idInt, ok2 := parseMediaID(idRaw)
		if !ok2 {
			continue
		}
		if _, existed := prevIDs[idInt]; existed {
			continue
		}

		// New item detected — trigger TMDB search + enqueue downloads in background
		go processNewMediaExtras(mediaType, idInt, cfg)
	}
}

// processNewMediaExtras fetches TMDB extras, marks downloaded state and enqueues downloads according to config.
func processNewMediaExtras(mediaType MediaType, mediaID int, cfg interface{}) {
	TrailarrLog(INFO, "SyncMediaCache", "New media detected, triggering extras search: mediaType=%v, id=%d", mediaType, mediaID)
	extras, err := FetchTMDBExtrasForMedia(mediaType, mediaID)
	if err != nil {
		TrailarrLog(WARN, "SyncMediaCache", "Failed to fetch TMDB extras for mediaType=%v id=%d: %v", mediaType, mediaID, err)
		return
	}
	cacheFile, _ := resolveCachePath(mediaType)
	mediaPath, _ := FindMediaPathByID(cacheFile, mediaID)
	MarkDownloadedExtras(extras, mediaPath, "type", "title")

	// Ensure cfg is the expected ExtraTypesConfig type before calling filterAndDownloadExtras.
	// If it's not present or of wrong type, fall back to zero value (defaults).
	var etcfg ExtraTypesConfig
	if cfg != nil {
		if v, ok := cfg.(ExtraTypesConfig); ok {
			etcfg = v
		} else {
			TrailarrLog(WARN, "SyncMediaCache", "Invalid extras config type; using defaults")
		}
	}
	filterAndDownloadExtras(mediaType, mediaID, extras, etcfg)
}

// Helper: fetch provider items and decode JSON, with logging preserved
func fetchProviderItems(provider, apiPath string) ([]map[string]interface{}, error) {
	providerURL, apiKey, err := GetProviderUrlAndApiKey(provider)
	if err != nil {
		return nil, fmt.Errorf("%s settings not found: %w", provider, err)
	}
	req, err := http.NewRequest("GET", providerURL+apiPath, nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error fetching %s: %w", provider, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		TrailarrLog(WARN, "SyncMediaCache", "%s API error: %d", provider, resp.StatusCode)
		return nil, fmt.Errorf("%s API error: %d", provider, resp.StatusCode)
	}
	var allItems []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allItems); err != nil {
		return nil, fmt.Errorf("failed to decode %s response: %w", provider, err)
	}
	return allItems, nil
}

// Helper: scan extras for a single item and persist them to the unified extras collection
func addExtrasFromItem(ctx context.Context, provider string, m map[string]interface{}) error {
	mediaPath, _ := m["path"].(string)
	extrasByType := scanExtrasInfo(mediaPath)
	mediaType := MediaTypeMovie
	if provider == "sonarr" {
		mediaType = MediaTypeTV
	}
	mediaId, _ := parseMediaID(m["id"])
	for extraType, extras := range extrasByType {
		for _, extra := range extras {
			title, _ := extra["Title"].(string)
			fileName, _ := extra["FileName"].(string)
			youtubeId, _ := extra["YoutubeId"].(string)
			status, _ := extra["Status"].(string)
			if status == "" {
				status = "downloaded"
			}
			entry := ExtrasEntry{
				MediaType:  mediaType,
				MediaId:    mediaId,
				ExtraTitle: title,
				ExtraType:  extraType,
				FileName:   fileName,
				YoutubeId:  youtubeId,
				Status:     status,
			}
			_ = AddOrUpdateExtra(ctx, entry)
		}
	}
	return nil
}

// Generic background sync for Radarr/Sonarr

// Returns a Gin handler to list media (movies/series) without any downloaded trailer extra
func GetMissingExtrasHandler(wantedPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := loadCache(wantedPath)
		if err != nil {
			respondError(c, http.StatusInternalServerError, "wanted cache not found")
			return
		}
		// Get required extra types from config
		cfg, err := GetExtraTypesConfig()
		if err != nil {
			respondError(c, http.StatusInternalServerError, "failed to load extra types config")
			return
		}
		// Map config keys to canonical Plex type names
		requiredTypes := GetEnabledCanonicalExtraTypes(cfg)
		TrailarrLog(INFO, "GetMissingExtrasHandler", "Required extra types: %v", requiredTypes)
		mediaType, _ := detectMediaTypeAndMainCachePath(wantedPath)
		missing := Filter(items, func(media map[string]interface{}) bool {
			id := media["id"]
			var mediaId int
			switch v := id.(type) {
			case float64:
				mediaId = int(v)
			case int:
				mediaId = v
			default:
				return true // treat as missing if no id
			}
			return !HasAnyEnabledExtras(mediaType, mediaId, requiredTypes)
		})
		TrailarrLog(INFO, "GetMissingExtrasHandler", "Found %d items missing extras of types: %v", len(missing), requiredTypes)
		respondJSON(c, http.StatusOK, gin.H{"items": missing})
	}
}

// sharedExtrasHandler handles extras for both movies and series
func sharedExtrasHandler(mediaType MediaType) gin.HandlerFunc {
	return func(c *gin.Context) {
		idStr := c.Param("id")
		var id int
		fmt.Sscanf(idStr, "%d", &id)

		// 1. Load persistent extras
		extras, err := SearchExtras(mediaType, id)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}

		// 2. Load TMDB extras (best-effort)
		tmdbExtras, err := FetchTMDBExtrasForMedia(mediaType, id)
		if err != nil {
			TrailarrLog(WARN, "sharedExtrasHandler", "Failed to fetch TMDB extras: %v", err)
			tmdbExtras = nil
		}

		// 3. Merge sources with persistent taking precedence
		finalExtras := mergeExtrasPrioritizePersistent(extras, tmdbExtras)

		// 4. Mark downloaded extras
		cacheFile, _ := resolveCachePath(mediaType)
		mediaPath, err := FindMediaPathByID(cacheFile, id)
		if err != nil {
			respondError(c, http.StatusInternalServerError, fmt.Sprintf("%s cache not found", mediaType))
			return
		}
		MarkDownloadedExtras(finalExtras, mediaPath, "type", "title")

		// 5. Apply rejected extras (preserve reason and include missing rejected entries)
		rejectedExtras := GetRejectedExtrasForMedia(mediaType, id)
		TrailarrLog(DEBUG, "sharedExtrasHandler", "Rejected extras: %+v", rejectedExtras)
		finalExtras = applyRejectedExtras(finalExtras, rejectedExtras)

		respondJSON(c, http.StatusOK, gin.H{"extras": finalExtras})
	}
}

// mergeExtrasPrioritizePersistent merges persistent and TMDB extras using YoutubeId+ExtraType+ExtraTitle as key,
// giving priority to persistent entries when duplicates exist.
func mergeExtrasPrioritizePersistent(persistent, tmdb []Extra) []Extra {
	allMap := make(map[string]Extra)
	keyFor := func(e Extra) string {
		return e.YoutubeId + ":" + e.ExtraType + ":" + e.ExtraTitle
	}
	for _, e := range persistent {
		allMap[keyFor(e)] = e
	}
	for _, e := range tmdb {
		k := keyFor(e)
		if _, exists := allMap[k]; !exists {
			allMap[k] = e
		}
	}
	result := make([]Extra, 0, len(allMap))
	for _, e := range allMap {
		result = append(result, e)
	}
	return result
}

// applyRejectedExtras updates finalExtras with rejected statuses and includes rejected entries not already present.
func applyRejectedExtras(finalExtras []Extra, rejectedExtras []RejectedExtra) []Extra {
	youtubeInFinal := make(map[string]struct{}, len(finalExtras))
	for i := range finalExtras {
		youtubeInFinal[finalExtras[i].YoutubeId] = struct{}{}
	}

	// Map of youtubeId -> reason for quick lookup
	rejectedReason := make(map[string]string, len(rejectedExtras))
	for _, r := range rejectedExtras {
		rejectedReason[r.YoutubeId] = r.Reason
	}

	// Apply reasons to existing extras
	for i := range finalExtras {
		if reason, ok := rejectedReason[finalExtras[i].YoutubeId]; ok {
			finalExtras[i].Status = "rejected"
			finalExtras[i].Reason = reason
		}
	}

	// Append rejected extras that are not present in finalExtras
	for _, r := range rejectedExtras {
		if _, exists := youtubeInFinal[r.YoutubeId]; !exists {
			finalExtras = append(finalExtras, Extra{
				ExtraType:  r.ExtraType,
				ExtraTitle: r.ExtraTitle,
				YoutubeId:  r.YoutubeId,
				Status:     "rejected",
				Reason:     r.Reason,
			})
		}
	}
	return finalExtras
}

// respondError is a helper for Gin error responses
func respondError(c *gin.Context, code int, msg string) {
	c.JSON(code, gin.H{"error": msg})
}

// respondJSON is a helper for Gin JSON responses
func respondJSON(c *gin.Context, code int, obj interface{}) {
	c.JSON(code, obj)
}

// Updates the main JSON file to mark items as wanted if they have no trailer
func updateWantedStatusInMainJson(mediaType MediaType, cacheFile string) error {
	items, err := loadCache(cacheFile)
	if err != nil {
		return err
	}
	for _, item := range items {
		id := item["id"]
		var mediaId int
		switch v := id.(type) {
		case float64:
			mediaId = int(v)
		case int:
			mediaId = v
		default:
			item["wanted"] = false
			continue
		}
		// Query persistent extras collection for this media item
		extras, _ := SearchExtras(mediaType, mediaId)
		hasTrailer := false
		for _, e := range extras {
			if strings.EqualFold(e.ExtraType, "Trailer") {
				hasTrailer = true
				break
			}
		}
		item["wanted"] = !hasTrailer
	}
	// Save to store for movies/series, file for others
	if cacheFile == MoviesStoreKey || cacheFile == SeriesStoreKey {
		return SaveMediaToStore(cacheFile, items)
	}
	return WriteJSONFile(cacheFile, items)
}

// Handler to get a single media item by path parameter (e.g. /api/movies/:id)
func GetMediaByIdHandler(cacheFile, key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		idParam := c.Param("id")
		TrailarrLog(DEBUG, "GetMediaByIdHandler", "HTTP %s %s, idParam: %s", c.Request.Method, c.Request.URL.String(), idParam)
		items, err := loadCache(cacheFile)
		if err != nil {
			TrailarrLog(DEBUG, "GetMediaByIdHandler", "Failed to load cache: %v", err)
			respondError(c, http.StatusInternalServerError, "cache not found")
			TrailarrLog(INFO, "GetMediaByIdHandler", totalTimeLogFormat, time.Since(start))
			return
		}
		filtered := Filter(items, func(m map[string]interface{}) bool {
			id, ok := m[key]
			return ok && fmt.Sprintf("%v", id) == idParam
		})
		TrailarrLog(DEBUG, "GetMediaByIdHandler", "Filtered by id=%s, %d items remain", idParam, len(filtered))
		if len(filtered) == 0 {
			respondError(c, http.StatusNotFound, "item not found")
			TrailarrLog(INFO, "GetMediaByIdHandler", totalTimeLogFormat, time.Since(start))
			return
		}
		TrailarrLog(DEBUG, "GetMediaByIdHandler", "Item: %+v", filtered[0])
		respondJSON(c, http.StatusOK, gin.H{"item": filtered[0]})
		TrailarrLog(INFO, "GetMediaByIdHandler", totalTimeLogFormat, time.Since(start))
	}
}

// Returns true if the media has any extras of the enabled types (case/plural robust)
func HasAnyEnabledExtras(mediaType MediaType, mediaId int, enabledTypes []string) bool {
	extras, _ := SearchExtras(mediaType, mediaId)
	for _, e := range extras {
		for _, typ := range enabledTypes {
			if strings.EqualFold(e.ExtraType, typ) || strings.EqualFold(e.ExtraType+"s", typ) || strings.EqualFold(e.ExtraType, typ+"s") {
				return true
			}
		}
	}
	return false
}

// SyncMediaType syncs Radarr or Sonarr depending on mediaType
func SyncMediaType(mediaType MediaType) error {
	switch mediaType {
	case MediaTypeMovie:
		return SyncMedia(
			"radarr",
			"/api/v3/movie",
			MoviesStoreKey,
			func(m map[string]interface{}) bool {
				hasFile, ok := m["hasFile"].(bool)
				return ok && hasFile
			},
			MediaCoverPath+"/Movies",
			[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
		)
	case MediaTypeTV:
		return SyncMedia(
			"sonarr",
			"/api/v3/series",
			SeriesStoreKey,
			func(m map[string]interface{}) bool {
				stats, ok := m["statistics"].(map[string]interface{})
				if !ok {
					return false
				}
				episodeFileCount, ok := stats["episodeFileCount"].(float64)
				return ok && episodeFileCount >= 1
			},
			MediaCoverPath+"/Series",
			[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
		)
	default:
		return fmt.Errorf("unknown media type: %v", mediaType)
	}
}
