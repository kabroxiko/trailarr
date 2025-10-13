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

// ProxyYouTubeImageHandler proxies YouTube thumbnail images to avoid 404s and CORS issues
func ProxyYouTubeImageHandler(c *gin.Context) {
	youtubeId := c.Param("youtubeId")
	if youtubeId == "" {
		respondError(c, http.StatusBadRequest, "Missing youtubeId")
		return
	}
	// Use MediaCoverPath/YouTube as cache directory
	cacheDir := filepath.Join(MediaCoverPath, "YouTube")
	// Ensure top-level MediaCoverPath exists
	if err := os.MkdirAll(MediaCoverPath, 0775); err != nil {
		TrailarrLog(WARN, "ProxyYouTubeImageHandler", "Failed to create MediaCoverPath %s: %v", MediaCoverPath, err)
	}
	// Ensure cacheDir exists
	if err := os.MkdirAll(cacheDir, 0775); err != nil {
		TrailarrLog(WARN, "ProxyYouTubeImageHandler", "Failed to create cacheDir %s: %v", cacheDir, err)
	}

	// Check for existing cached files with common extensions
	exts := []string{".jpg", ".jpeg", ".png", ".webp", ".svg"}
	var cachedPath string
	var contentType string
	for _, e := range exts {
		p := filepath.Join(cacheDir, youtubeId+e)
		if _, err := os.Stat(p); err == nil {
			cachedPath = p
			switch e {
			case ".jpg", ".jpeg":
				contentType = "image/jpeg"
			case ".png":
				contentType = "image/png"
			case ".webp":
				contentType = "image/webp"
			case ".svg":
				contentType = "image/svg+xml"
			default:
				contentType = "application/octet-stream"
			}
			break
		}
	}
	if cachedPath != "" {
		// Serve cached file
		c.Header("Content-Type", contentType)
		c.Header("Cache-Control", "public, max-age=86400")
		c.File(cachedPath)
		return
	}

	// Not cached — try to fetch from YouTube image servers
	thumbUrls := []string{
		"https://i.ytimg.com/vi/" + youtubeId + "/maxresdefault.jpg",
		"https://i.ytimg.com/vi/" + youtubeId + "/hqdefault.jpg",
	}
	var resp *http.Response
	var err error
	for _, url := range thumbUrls {
		resp, err = http.Get(url)
		if err == nil && resp.StatusCode == 200 {
			break
		}
		if resp != nil {
			resp.Body.Close()
		}
	}
	if err != nil || resp == nil || resp.StatusCode != 200 {
		// Serve a small inline SVG fallback that looks like FontAwesome's faBan
		svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 128 128" width="64" height="64" role="img" aria-label="Unavailable">
  <circle cx="64" cy="64" r="30" fill="none" stroke="#888" stroke-width="8" />
  <!-- diagonal from top-right to bottom-left -->
  <line x1="92" y1="36" x2="36" y2="92" stroke="#888" stroke-width="10" stroke-linecap="round" />
</svg>`
		c.Header("Content-Type", "image/svg+xml")
		// Indicate to the frontend that this response is a fallback placeholder
		c.Header("X-Proxy-Fallback", "1")
		c.Header("Cache-Control", "public, max-age=86400")
		c.Status(http.StatusOK)
		// If this is a HEAD request, return headers only (no body)
		if c.Request.Method == http.MethodHead {
			return
		}
		_, _ = c.Writer.Write([]byte(svg))
		return
	}
	defer resp.Body.Close()

	// Determine extension from content-type
	ct := resp.Header.Get("Content-Type")
	ext := ".jpg"
	switch {
	case strings.Contains(ct, "jpeg"):
		ext = ".jpg"
	case strings.Contains(ct, "png"):
		ext = ".png"
	case strings.Contains(ct, "webp"):
		ext = ".webp"
	case strings.Contains(ct, "svg"):
		ext = ".svg"
	}

	tmpPath := filepath.Join(cacheDir, youtubeId+".tmp")
	finalPath := filepath.Join(cacheDir, youtubeId+ext)
	out, err := os.Create(tmpPath)
	if err != nil {
		// Can't cache — just stream response
		c.Header("Content-Type", ct)
		c.Header("Cache-Control", "public, max-age=86400")
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, resp.Body)
		return
	}
	_, _ = io.Copy(out, resp.Body)
	out.Close()
	// Rename tmp to final
	_ = os.Rename(tmpPath, finalPath)

	// Serve the saved file
	c.Header("Content-Type", ct)
	c.Header("Cache-Control", "public, max-age=86400")
	// If HEAD request, return headers only
	if c.Request.Method == http.MethodHead {
		c.Status(http.StatusOK)
		return
	}
	c.File(finalPath)
}

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)

// Syncs media cache and caches poster images for Radarr/Sonarr
func SyncMedia(provider, apiPath, cacheFile string, filter func(map[string]interface{}) bool, posterDir string, posterSuffixes []string) error {
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
	if path == MoviesJSONPath || path == SeriesJSONPath {
		items, err := LoadMediaFromRedis(path)
		if err != nil {
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
				// Attach extras from collection
				mediaPath, _ := item["path"].(string)
				if mediaPath != "" {
					extrasRaw := scanExtrasInfo(mediaPath)
					extras := make(map[string]interface{})
					for k, v := range extrasRaw {
						extras[k] = v
					}
					item["extras"] = extras
				} else {
					item["extras"] = map[string]interface{}{}
				}
			}
		}
		return items, nil
	}
	// Fallback to file for other paths
	var items []map[string]interface{}
	if err := ReadJSONFile(path, &items); err != nil {
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
			// Attach extras from collection
			mediaPath, _ := item["path"].(string)
			if mediaPath != "" {
				extras := scanExtrasInfo(mediaPath)
				item["extras"] = extras
			} else {
				item["extras"] = map[string]interface{}{}
			}
		}
	}
	return items, nil
}

// LoadMediaFromRedis loads movies or series from Redis, expects path to be MoviesJSONPath or SeriesJSONPath
func LoadMediaFromRedis(path string) ([]map[string]interface{}, error) {
	client := GetRedisClient()
	ctx := context.Background()
	var redisKey string
	switch path {
	case MoviesJSONPath:
		redisKey = "trailarr:movies"
	case SeriesJSONPath:
		redisKey = "trailarr:series"
	default:
		return nil, fmt.Errorf("unsupported path for redis: %s", path)
	}
	val, err := client.Get(ctx, redisKey).Result()
	if err != nil {
		if err.Error() == "redis: nil" {
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

// SaveMediaToRedis saves movies or series to Redis, expects path to be MoviesJSONPath or SeriesJSONPath
func SaveMediaToRedis(path string, items []map[string]interface{}) error {
	client := GetRedisClient()
	ctx := context.Background()
	var redisKey string
	switch path {
	case MoviesJSONPath:
		redisKey = "trailarr:movies"
	case SeriesJSONPath:
		redisKey = "trailarr:series"
	default:
		return fmt.Errorf("unsupported path for redis: %s", path)
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
func SyncMediaCacheJson(provider, apiPath, cacheFile string, filter func(map[string]interface{}) bool) error {
	providerURL, apiKey, err := GetProviderUrlAndApiKey(provider)
	if err != nil {
		return fmt.Errorf("%s settings not found: %w", provider, err)
	}
	req, err := http.NewRequest("GET", providerURL+apiPath, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching %s: %w", provider, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		TrailarrLog(WARN, "SyncMediaCacheJson", "%s API error: %d", provider, resp.StatusCode)
		return fmt.Errorf("%s API error: %d", provider, resp.StatusCode)
	}
	var allItems []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allItems); err != nil {
		return fmt.Errorf("failed to decode %s response: %w", provider, err)
	}
	items := make([]map[string]interface{}, 0)
	ctx := context.Background()
	for _, m := range allItems {
		if filter(m) {
			mediaPath, _ := m["path"].(string)
			extrasByType := scanExtrasInfo(mediaPath)
			// Only record extras in the unified collection, not in the per-media cache
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
			items = append(items, m)
		}
	}
	// Save to Redis for movies/series, file for others
	if cacheFile == MoviesJSONPath || cacheFile == SeriesJSONPath {
		_ = SaveMediaToRedis(cacheFile, items)
	} else {
		_ = WriteJSONFile(cacheFile, items)
	}
	TrailarrLog(INFO, "SyncMediaCacheJson", "[Sync%s] Synced %d items to cache.", provider, len(items))

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
		missing := Filter(items, func(media map[string]interface{}) bool {
			return !HasAnyEnabledExtras(media, requiredTypes)
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
		// Set status to "rejected" for any extra whose YoutubeId matches a rejected extra
		rejectedYoutubeIds := make(map[string]struct{})
		for _, rejected := range rejectedExtras {
			rejectedYoutubeIds[rejected.YoutubeId] = struct{}{}
		}
		MarkRejectedExtras(extras, rejectedYoutubeIds)
		// Also append any rejected extras not already present in extras
		for _, rejected := range rejectedExtras {
			if _, exists := youtubeIdInResults[rejected.YoutubeId]; !exists {
				extras = append(extras, Extra{
					ExtraType:  rejected.ExtraType,
					ExtraTitle: rejected.ExtraTitle,
					YoutubeId:  rejected.YoutubeId,
					Status:     "rejected",
				})
			}
		}
		TrailarrLog(DEBUG, "sharedExtrasHandler", "Final extras: %+v", extras)
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

// Updates the main JSON file to mark items as wanted if they have no trailer
func updateWantedStatusInMainJson(mediaType MediaType, cacheFile string) error {
	items, err := loadCache(cacheFile)
	if err != nil {
		return err
	}
	mappings, err := GetPathMappings(mediaType)
	if err != nil {
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
	for _, item := range items {
		path, ok := item["path"].(string)
		if !ok {
			item["wanted"] = false
			continue
		}
		item["wanted"] = !trailerSet[path]
	}
	// Save to Redis for movies/series, file for others
	if cacheFile == MoviesJSONPath || cacheFile == SeriesJSONPath {
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
			TrailarrLog(INFO, "GetMediaByIdHandler", "Total time: %v", time.Since(start))
			return
		}
		filtered := Filter(items, func(m map[string]interface{}) bool {
			id, ok := m[key]
			return ok && fmt.Sprintf("%v", id) == idParam
		})
		TrailarrLog(DEBUG, "GetMediaByIdHandler", "Filtered by id=%s, %d items remain", idParam, len(filtered))
		if len(filtered) == 0 {
			respondError(c, http.StatusNotFound, "item not found")
			TrailarrLog(INFO, "GetMediaByIdHandler", "Total time: %v", time.Since(start))
			return
		}
		TrailarrLog(DEBUG, "GetMediaByIdHandler", "Item: %+v", filtered[0])
		respondJSON(c, http.StatusOK, gin.H{"item": filtered[0]})
		TrailarrLog(INFO, "GetMediaByIdHandler", "Total time: %v", time.Since(start))
	}
}

// Returns true if the media has any extras of the enabled types (case/plural robust)
func HasAnyEnabledExtras(media map[string]interface{}, enabledTypes []string) bool {
	extras, ok := media["extras"]
	if !ok {
		return false
	}
	extrasMap, ok := extras.(map[string]interface{})
	if !ok {
		return false
	}
	lowerKeys := make(map[string]string)
	for k := range extrasMap {
		lowerKeys[strings.ToLower(k)] = k
	}
	for _, typ := range enabledTypes {
		tLower := strings.ToLower(typ)
		key, found := lowerKeys[tLower]
		if !found && strings.HasSuffix(tLower, "s") {
			key, found = lowerKeys[tLower[:len(tLower)-1]]
		}
		if !found && !strings.HasSuffix(tLower, "s") {
			key, found = lowerKeys[tLower+"s"]
		}
		if found {
			v := extrasMap[key]
			switch vv := v.(type) {
			case []interface{}:
				if len(vv) > 0 {
					return true
				}
			case string:
				if strings.TrimSpace(vv) != "" {
					return true
				}
			case nil:
				// skip nil
			default:
				// Only return true for non-empty values
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
			MoviesJSONPath,
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
			SeriesJSONPath,
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
