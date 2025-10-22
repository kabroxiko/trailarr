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
	err := SyncMediaCache(provider, apiPath, cacheFile, filter)
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
	for _, item := range idList {
		id := fmt.Sprintf("%v", item[idKey])
		settings, err := loadMediaSettings(section)
		if err != nil {
			continue
		}
		apiBase := trimTrailingSlash(settings.ProviderURL)
		jobs := Map(posterSuffixes, func(suffix string) posterJob {
			idDir := baseDir + "/" + id
			localPath := idDir + suffix
			posterUrl := apiBase + RemoteMediaCoverPath + id + suffix
			return posterJob{id, idDir, localPath, posterUrl}
		})
		for _, job := range jobs {
			if err := os.MkdirAll(job.idDir, 0775); err != nil {
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
	// Use Redis for movies and series
	if path == MoviesRedisKey || path == SeriesRedisKey {
		items, err := LoadMediaFromRedis(path)
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

// LoadMediaFromRedis loads movies or series from Redis, expects path to be MoviesRedisKey or SeriesRedisKey
func LoadMediaFromRedis(path string) ([]map[string]interface{}, error) {
	client := GetRedisClient()
	ctx := context.Background()
	var redisKey string
	switch path {
	case MoviesRedisKey:
		redisKey = "trailarr:movies"
	case SeriesRedisKey:
		redisKey = "trailarr:series"
	default:
		return nil, fmt.Errorf("unsupported path for bbolt: %s", path)
	}
	valRes := client.Get(ctx, redisKey)
	val, err := valRes.Result()
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

// SaveMediaToRedis saves movies or series to Redis, expects path to be MoviesRedisKey or SeriesRedisKey
func SaveMediaToRedis(path string, items []map[string]interface{}) error {
	client := GetRedisClient()
	ctx := context.Background()
	var redisKey string
	switch path {
	case MoviesRedisKey:
		redisKey = "trailarr:movies"
	case SeriesRedisKey:
		redisKey = "trailarr:series"
	default:
		return fmt.Errorf("unsupported path for bbolt: %s", path)
	}
	data, err := json.Marshal(items)
	if err != nil {
		return err
	}
	return client.Set(ctx, redisKey, data, 0).Err()
}

// Helper: Detect media type and main cache path
func detectMediaTypeAndMainCachePath(path string) (MediaType, string) {
	if strings.Contains(path, "movie") || strings.Contains(path, "Movie") {
		return MediaTypeMovie, MoviesRedisKey
	} else if strings.Contains(path, "series") || strings.Contains(path, "Series") {
		return MediaTypeTV, SeriesRedisKey
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
	allItems, err := fetchProviderItems(provider, apiPath)
	if err != nil {
		return err
	}

	ctx := context.Background()
	items := collectFilteredItems(ctx, provider, allItems, filter)

	// Load previous items (if any) so we can detect newly added media
	prevItems, _ := loadCache(cacheFile)

	// Save items to the appropriate backend
	_ = saveItems(cacheFile, items)

	// Handle new items (best-effort, background tasks)
	handleNewItems(provider, items, prevItems)

	TrailarrLog(INFO, "SyncMediaCache", "[Sync%s] Synced %d items to cache.", provider, len(items))

	// After syncing main cache, update wanted status in main JSON
	var mediaType MediaType
	if provider == "radarr" {
		mediaType = MediaTypeMovie
	} else {
		mediaType = MediaTypeTV
	}
	_ = updateWantedStatusInMainJson(mediaType, cacheFile)
	return nil
}

// collectFilteredItems applies the filter and records extras for each accepted item.
func collectFilteredItems(ctx context.Context, provider string, allItems []map[string]interface{}, filter func(map[string]interface{}) bool) []map[string]interface{} {
	items := make([]map[string]interface{}, 0)
	for _, m := range allItems {
		if !filter(m) {
			continue
		}
		// Record extras into the unified collection (not attached to the item)
		_ = addExtrasFromItem(ctx, provider, m)
		items = append(items, m)
	}
	return items
}

// saveItems persists items either to Redis or to a file depending on cacheFile.
func saveItems(cacheFile string, items []map[string]interface{}) error {
	if cacheFile == MoviesRedisKey || cacheFile == SeriesRedisKey {
		return SaveMediaToRedis(cacheFile, items)
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
	// Save to Redis for movies/series, file for others
	if cacheFile == MoviesRedisKey || cacheFile == SeriesRedisKey {
		return SaveMediaToRedis(cacheFile, items)
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
			MoviesRedisKey,
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
			SeriesRedisKey,
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
