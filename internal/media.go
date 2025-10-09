package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

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
		apiBase := trimTrailingSlash(settings.URL)
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
	url, apiKey, err := GetProviderUrlAndApiKey(provider)
	if err != nil {
		return fmt.Errorf("%s settings not found: %w", provider, err)
	}
	req, err := http.NewRequest("GET", url+apiPath, nil)
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
	for _, m := range allItems {
		if filter(m) {
			// Add existing_extras info
			mediaPath, _ := m["path"].(string)
			m["existing_extras"] = scanExtrasInfo(mediaPath)
			items = append(items, m)
		}
	}
	_ = WriteJSONFile(cacheFile, items)
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
		missing := Filter(items, func(m map[string]interface{}) bool {
			return !HasAnyEnabledExtras(m, requiredTypes)
		})
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
	return WriteJSONFile(cacheFile, items)
}

// Handler to get a single media item by path parameter (e.g. /api/movies/:id)
func GetMediaByIdHandler(cacheFile, key string) gin.HandlerFunc {
	return func(c *gin.Context) {
		idParam := c.Param("id")
		TrailarrLog(DEBUG, "GetMediaByIdHandler", "HTTP %s %s, idParam: %s", c.Request.Method, c.Request.URL.String(), idParam)
		items, err := loadCache(cacheFile)
		if err != nil {
			TrailarrLog(DEBUG, "GetMediaByIdHandler", "Failed to load cache: %v", err)
			respondError(c, http.StatusInternalServerError, "cache not found")
			return
		}
		filtered := Filter(items, func(m map[string]interface{}) bool {
			id, ok := m[key]
			return ok && fmt.Sprintf("%v", id) == idParam
		})
		TrailarrLog(DEBUG, "GetMediaByIdHandler", "Filtered by id=%s, %d items remain", idParam, len(filtered))
		if len(filtered) == 0 {
			respondError(c, http.StatusNotFound, "item not found")
			return
		}
		TrailarrLog(DEBUG, "GetMediaByIdHandler", "Item: %+v", filtered[0])
		respondJSON(c, http.StatusOK, gin.H{"item": filtered[0]})
	}
}

// Returns true if the item has any extras of the enabled types (case/plural robust)
func HasAnyEnabledExtras(item map[string]interface{}, enabledTypes []string) bool {
	extras, ok := item["existing_extras"]
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
		} else if !found {
			key, found = lowerKeys[tLower+"s"]
		}
		if found {
			v := extrasMap[key]
			switch vv := v.(type) {
			case []interface{}:
				if len(vv) > 0 {
					return true
				}
			default:
				return true
			}
		}
	}
	return false
}
