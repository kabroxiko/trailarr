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
	"gopkg.in/yaml.v3"
)

// Helper to fetch and cache poster image
func fetchAndCachePoster(localPath, posterUrl, section string) error {
	resp, err := http.Get(posterUrl)
	if err != nil || resp.StatusCode != 200 {
		if resp != nil {
			resp.Body.Close()
		}
		return fmt.Errorf("Failed to fetch poster image from %s", section)
	}
	defer resp.Body.Close()
	out, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("Failed to cache poster image for %s", section)
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
	for _, item := range idList {
		id := fmt.Sprintf("%v", item[idKey])
		for _, suffix := range posterSuffixes {
			idDir := baseDir + "/" + id
			if err := os.MkdirAll(idDir, 0775); err != nil {
				continue
			}
			localPath := idDir + suffix
			if _, err := os.Stat(localPath); err == nil {
				continue
			}
			settings, err := loadMediaSettings(section)
			if err != nil {
				continue
			}
			apiBase := trimTrailingSlash(settings.URL)
			posterUrl := apiBase + RemoteMediaCoverPath + id + suffix
			_ = fetchAndCachePoster(localPath, posterUrl, section)
		}
	}
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

// Generic poster handler for Radarr and Sonarr
func getImageHandler(section string, idParam string, fileSuffix string) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param(idParam)
		settings, err := loadMediaSettings(section)
		if err != nil {
			c.String(500, "Invalid %s settings", section)
			return
		}
		apiBase := trimTrailingSlash(settings.URL)
		var localPath string
		switch section {
		case "radarr":
			localPath = MediaCoverPath + "Movies/" + id + fileSuffix
		case "sonarr":
			localPath = MediaCoverPath + "Series/" + id + fileSuffix
		default:
			localPath = MediaCoverPath + id + fileSuffix
		}
		if _, err := os.Stat(localPath); err == nil {
			c.File(localPath)
			return
		}
		posterUrl := apiBase + RemoteMediaCoverPath + id + fileSuffix
		if err := fetchAndCachePoster(localPath, posterUrl, section); err == nil {
			c.File(localPath)
			return
		}
		// If can't cache, just proxy
		resp, err := http.Get(posterUrl)
		if err != nil || resp.StatusCode != 200 {
			c.String(502, "Failed to fetch poster image from %s", section)
			return
		}
		defer resp.Body.Close()
		c.Header(HeaderContentType, resp.Header.Get(HeaderContentType))
		c.Status(http.StatusOK)
		_, _ = io.Copy(c.Writer, resp.Body)
	}
}

// Common settings struct for both Radarr and Sonarr
// Use this for loading settings generically
type MediaSettings struct {
	URL    string `yaml:"url"`
	APIKey string `yaml:"apiKey"`
}

// Loads settings for a given section ("radarr" or "sonarr")
func loadMediaSettings(section string) (MediaSettings, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return MediaSettings{}, fmt.Errorf("settings not found: %w", err)
	}
	var allSettings map[string]map[string]string
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		return MediaSettings{}, fmt.Errorf("invalid settings: %w", err)
	}
	sec, ok := allSettings[section]
	if !ok {
		return MediaSettings{}, fmt.Errorf("section %s not found", section)
	}
	return MediaSettings{URL: sec["url"], APIKey: sec["apiKey"]}, nil
}

// Trims trailing slash from a URL
func trimTrailingSlash(url string) string {
	if strings.HasSuffix(url, "/") {
		return strings.TrimRight(url, "/")
	}
	return url
}

// Proxies an image from a remote API, optionally setting API key header
func proxyImage(c *gin.Context, imageUrl, apiBase, apiKey string) error {
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		return fmt.Errorf("Error creating image request")
	}
	if strings.HasPrefix(imageUrl, apiBase) {
		req.Header.Set(HeaderApiKey, apiKey)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("Failed to fetch image")
	}
	defer resp.Body.Close()
	c.Header(HeaderContentType, resp.Header.Get(HeaderContentType))
	c.Status(http.StatusOK)
	_, copyErr := io.Copy(c.Writer, resp.Body)
	return copyErr
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

	// Determine section and get pathMappings
	var section string
	if strings.Contains(path, "movie") || strings.Contains(path, "Movie") {
		section = "radarr"
	} else if strings.Contains(path, "series") || strings.Contains(path, "Series") {
		section = "sonarr"
	}
	if section != "" {
		data, err := os.ReadFile(ConfigPath)
		if err == nil {
			var config map[string]interface{}
			if yaml.Unmarshal(data, &config) == nil {
				sec, _ := config[section].(map[string]interface{})
				var mappings [][2]string
				if sec != nil {
					if pm, ok := sec["pathMappings"].([]interface{}); ok {
						for _, m := range pm {
							if mMap, ok := m.(map[string]interface{}); ok {
								from, _ := mMap["from"].(string)
								to, _ := mMap["to"].(string)
								if from != "" && to != "" {
									mappings = append(mappings, [2]string{from, to})
								}
							}
						}
					}
				}
				// For each item, convert root folder part of path
				for _, item := range items {
					p, ok := item["path"].(string)
					if !ok || p == "" {
						continue
					}
					for _, m := range mappings {
						if strings.HasPrefix(p, m[0]) {
							item["path"] = m[1] + p[len(m[0]):]
							break
						}
					}
				}
			}
		}
	}
	return items, nil
}

// Writes a generic slice to a JSON cache file
func writeCache(items []map[string]interface{}, path string) error {
	cacheData, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, cacheData, 0644)
}

// Move SyncMediaCacheJson to media.go for shared use
// Generic sync function for Radarr/Sonarr
// Syncs only the JSON cache for Radarr/Sonarr, not the media files themselves
// Pass section ("radarr" or "sonarr"), apiPath (e.g. "/api/v3/movie"), cachePath, and a filter function for items
func SyncMediaCacheJson(section, apiPath, cachePath string, filter func(map[string]interface{}) bool) error {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return fmt.Errorf("%s settings not found: %w", section, err)
	}
	var allSettings map[string]map[string]string
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		return fmt.Errorf("invalid %s settings: %w", section, err)
	}
	settings, ok := allSettings[section]
	if !ok {
		return fmt.Errorf("section %s not found in config", section)
	}
	url := settings["url"]
	apiKey := settings["apiKey"]
	req, err := http.NewRequest("GET", url+apiPath, nil)
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error fetching %s: %w", section, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return fmt.Errorf("%s API error: %d", section, resp.StatusCode)
	}
	var allItems []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allItems); err != nil {
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
	fmt.Printf("[Sync%s] Synced %d items to cache.\n", section, len(items))
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
