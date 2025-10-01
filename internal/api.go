package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/gin-gonic/gin"
)

const (
	ConfigPath               = "/var/lib/extrazarr/config/config.yml"
	MoviesCachePath          = "/var/lib/extrazarr/movies_cache.json"
	MediaCoverPath           = "/var/lib/extrazarr/MediaCover/"
	SeriesCachePath          = "/var/lib/extrazarr/series_cache.json"
	ErrInvalidSonarrSettings = "Invalid Sonarr settings"
	RemoteMediaCoverPath     = "/MediaCover/"
	HeaderApiKey             = "X-Api-Key"
	ErrInvalidRequest        = "invalid request"
	HeaderContentType        = "Content-Type"
)

// RegisterRoutes registers all API endpoints to the Gin router
func RegisterRoutes(r *gin.Engine) {
	// Serve static files for movie posters
	r.Static("/mediacover", "/var/lib/extrazarr/MediaCover")
	r.StaticFile("/logo.svg", "web/public/logo.svg")
	r.GET("/api/radarr/movies", getRadarrMoviesHandler)
	r.POST("/api/settings/radarr", saveRadarrSettingsHandler)
	r.GET("/api/settings/radarr", getRadarrSettingsHandler)
	r.GET("/api/extras/search", searchExtrasHandler)
	r.POST("/api/extras/download", downloadExtraHandler)
	r.GET("/api/extras/existing", existingExtrasHandler)
	r.GET("/api/sonarr/series", HandleSonarrSeries)
	r.POST("/api/settings/sonarr", saveSonarrSettingsHandler)
	r.GET("/api/settings/sonarr", getSonarrSettingsHandler)
	// Sonarr poster and banner proxy endpoints
	r.GET("/api/sonarr/poster/:seriesId", HandleSonarrPoster)
	r.GET("/api/sonarr/banner/:seriesId", HandleSonarrBanner)
	// Radarr poster and banner proxy endpoints
	r.GET("/api/radarr/poster/:movieId", HandleRadarrPoster)
	r.GET("/api/radarr/banner/:movieId", HandleRadarrBanner)
	// General settings (TMDB key)
	r.GET("/api/settings/general", getGeneralSettingsHandler)
	r.POST("/api/settings/general", saveGeneralSettingsHandler)
}

// Handler to get general settings (TMDB key)
func getGeneralSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"tmdbKey": ""})
		return
	}
	var allSettings struct {
		General struct {
			TMDBApiKey string `yaml:"tmdbKey"`
		} `yaml:"general"`
	}
	_ = yaml.Unmarshal(data, &allSettings)
	c.JSON(http.StatusOK, gin.H{"tmdbKey": allSettings.General.TMDBApiKey})
}

// Handler to save general settings (TMDB key)
func saveGeneralSettingsHandler(c *gin.Context) {
	var req struct {
		TMDBApiKey string `json:"tmdbKey" yaml:"tmdbKey"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	// Read existing settings
	var allSettings struct {
		General struct {
			TMDBApiKey string `yaml:"tmdbKey"`
		} `yaml:"general"`
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	data, _ := os.ReadFile(ConfigPath)
	_ = yaml.Unmarshal(data, &allSettings)
	allSettings.General.TMDBApiKey = req.TMDBApiKey
	out, _ := yaml.Marshal(allSettings)
	err := os.WriteFile(ConfigPath, out, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// Handler for /api/radarr/poster/:movieId
func HandleRadarrPoster(c *gin.Context) {
	movieId := c.Param("movieId")
	// Load Radarr settings
	data, err := os.ReadFile(ConfigPath)
	var allSettings struct {
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		c.String(http.StatusInternalServerError, "Invalid Radarr settings")
		return
	}
	radarrSettings := allSettings.Radarr
	// Remove trailing slash from URL if present
	apiBase := radarrSettings.URL
	if strings.HasSuffix(apiBase, "/") {
		apiBase = strings.TrimRight(apiBase, "/")
	}
	// Try local MediaCover first
	localPath := MediaCoverPath + movieId + "/poster-500.jpg"
	if _, err := os.Stat(localPath); err == nil {
		c.File(localPath)
		return
	}
	// Fallback to Radarr API
	posterUrl := apiBase + RemoteMediaCoverPath + movieId + "/poster-500.jpg"
	req, err := http.NewRequest("GET", posterUrl, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating poster request")
		return
	}
	req.Header.Set(HeaderApiKey, radarrSettings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch poster image from Radarr")
		return
	}
	defer resp.Body.Close()
	c.Header(HeaderContentType, resp.Header.Get(HeaderContentType))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}

// Handler for /api/radarr/banner/:movieId (fanart)
func HandleRadarrBanner(c *gin.Context) {
	movieId := c.Param("movieId")
	// Load Radarr settings
	data, err := os.ReadFile(ConfigPath)
	var allSettings struct {
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		c.String(http.StatusInternalServerError, "Invalid Radarr settings")
		return
	}
	radarrSettings := allSettings.Radarr
	// Remove trailing slash from URL if present
	apiBase := radarrSettings.URL
	if strings.HasSuffix(apiBase, "/") {
		apiBase = strings.TrimRight(apiBase, "/")
	}
	// Try local MediaCover first
	localPath := MediaCoverPath + movieId + "/fanart-1280.jpg"
	if _, err := os.Stat(localPath); err == nil {
		c.File(localPath)
		return
	}
	// Fallback to Radarr API
	bannerUrl := apiBase + RemoteMediaCoverPath + movieId + "/fanart-1280.jpg"
	req, err := http.NewRequest("GET", bannerUrl, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating banner request")
		return
	}
	req.Header.Set(HeaderApiKey, radarrSettings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch banner image from Radarr")
		return
	}
	defer resp.Body.Close()
	c.Header(HeaderContentType, resp.Header.Get(HeaderContentType))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}

// Handler for /api/sonarr/banner/:seriesId
func HandleSonarrBanner(c *gin.Context) {
	seriesId := c.Param("seriesId")
	// Load Sonarr settings
	data, err := os.ReadFile(ConfigPath)
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		c.String(http.StatusInternalServerError, ErrInvalidSonarrSettings)
		return
	}
	sonarrSettings := allSettings.Sonarr
	// Remove trailing slash from URL if present
	apiBase := sonarrSettings.URL
	if strings.HasSuffix(apiBase, "/") {
		apiBase = strings.TrimRight(apiBase, "/")
	}
	// Try local MediaCover first
	localPath := MediaCoverPath + seriesId + "/banner.jpg"
	if _, err := os.Stat(localPath); err == nil {
		c.File(localPath)
		return
	}
	// Fallback to Sonarr API
	bannerUrl := apiBase + RemoteMediaCoverPath + seriesId + "/banner.jpg"
	req, err := http.NewRequest("GET", bannerUrl, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating banner request")
		return
	}
	req.Header.Set(HeaderApiKey, sonarrSettings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch banner image from Sonarr")
		return
	}
	defer resp.Body.Close()
	c.Header(HeaderContentType, resp.Header.Get(HeaderContentType))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}

// Handler for /api/sonarr/poster/:seriesId
func HandleSonarrPoster(c *gin.Context) {
	seriesId := c.Param("seriesId")
	sonarrSettings, err := getSonarrSettings()
	if err != nil {
		c.String(http.StatusInternalServerError, ErrInvalidSonarrSettings)
		return
	}
	apiBase := trimTrailingSlash(sonarrSettings.URL)
	posterUrl, err := getSonarrSeriesPosterUrl(apiBase, sonarrSettings.APIKey, seriesId)
	if err != nil {
		c.String(http.StatusNotFound, err.Error())
		return
	}
	if err := proxyImage(c, posterUrl, apiBase, sonarrSettings.APIKey); err != nil {
		c.String(http.StatusBadGateway, err.Error())
	}
}

func getSonarrSettings() (struct {
	URL    string
	APIKey string
}, error) {
	data, err := os.ReadFile(ConfigPath)
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
	}
	if err != nil {
		return struct {
			URL    string
			APIKey string
		}{}, err
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		return struct {
			URL    string
			APIKey string
		}{}, err
	}
	return struct {
		URL    string
		APIKey string
	}{
		URL:    allSettings.Sonarr.URL,
		APIKey: allSettings.Sonarr.APIKey,
	}, nil
}

func trimTrailingSlash(url string) string {
	if strings.HasSuffix(url, "/") {
		return strings.TrimRight(url, "/")
	}
	return url
}

func getSonarrSeriesPosterUrl(apiBase, apiKey, seriesId string) (string, error) {
	req, err := http.NewRequest("GET", apiBase+"/api/v3/series/"+seriesId, nil)
	if err != nil {
		return "", fmt.Errorf("Error creating request")
	}
	req.Header.Set(HeaderApiKey, apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return "", fmt.Errorf("Failed to fetch series info from Sonarr")
	}
	defer resp.Body.Close()
	var series map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		return "", fmt.Errorf("Failed to decode Sonarr response")
	}
	images, ok := series["images"].([]interface{})
	if !ok {
		return "", fmt.Errorf("No poster found for series")
	}
	posterUrl, found := findPosterUrl(images, apiBase)
	if !found {
		return "", fmt.Errorf("No poster found for series")
	}
	return posterUrl, nil
}

func findPosterUrl(images []interface{}, apiBase string) (string, bool) {
	for _, img := range images {
		m, ok := img.(map[string]interface{})
		if !ok {
			continue
		}
		if m["coverType"] == "poster" {
			if remoteUrl, ok := m["remoteUrl"].(string); ok && remoteUrl != "" {
				return remoteUrl, true
			}
			if url, ok := m["url"].(string); ok && url != "" {
				return apiBase + url, true
			}
		}
	}
	return "", false
}

func proxyImage(c *gin.Context, imageUrl, apiBase, apiKey string) error {
	req, err := http.NewRequest("GET", imageUrl, nil)
	if err != nil {
		return fmt.Errorf("Error creating poster request")
	}
	if strings.HasPrefix(imageUrl, apiBase) {
		req.Header.Set(HeaderApiKey, apiKey)
	}
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		return fmt.Errorf("Failed to fetch poster image")
	}
	defer resp.Body.Close()
	c.Header(HeaderContentType, resp.Header.Get(HeaderContentType))
	c.Status(http.StatusOK)
	_, copyErr := io.Copy(c.Writer, resp.Body)
	return copyErr
}

// Handler to get Sonarr settings
func getSonarrSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": allSettings.Sonarr.URL, "apiKey": allSettings.Sonarr.APIKey})
}

// Handler to save Sonarr settings
func saveSonarrSettingsHandler(c *gin.Context) {
	var req struct {
		URL    string `yaml:"url"`
		APIKey string `yaml:"apiKey"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	// Read existing settings
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	data, _ := os.ReadFile(ConfigPath)
	_ = yaml.Unmarshal(data, &allSettings)
	allSettings.Sonarr.URL = req.URL
	allSettings.Sonarr.APIKey = req.APIKey
	out, _ := yaml.Marshal(allSettings)
	err := os.WriteFile(ConfigPath, out, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// --- Sonarr Series API ---
type SonarrSeries struct {
	ID    int    `json:"id"`
	Title string `json:"title"`
	Year  int    `json:"year"`
	Path  string `json:"path"`
}

// Handler for /api/sonarr/series
func HandleSonarrSeries(c *gin.Context) {
	cachePath := SeriesCachePath
	if series, ok := loadSonarrSeriesFromCache(cachePath); ok {
		c.JSON(http.StatusOK, gin.H{"series": series})
		return
	}

	sonarrSettings, err := getSonarrSettingsFromConfig()
	if err != nil {
		fmt.Println("[HandleSonarrSeries] Sonarr settings not found or invalid:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sonarr settings not found"})
		return
	}

	apiBase := trimTrailingSlash(sonarrSettings.URL)
	series, err := fetchAndFilterSonarrSeries(apiBase, sonarrSettings.APIKey)
	if err != nil {
		fmt.Println("[HandleSonarrSeries] Error fetching or decoding series:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cacheData, _ := json.MarshalIndent(series, "", "  ")
	_ = os.WriteFile(cachePath, cacheData, 0644)
	c.JSON(http.StatusOK, gin.H{"series": series})
}

func loadSonarrSeriesFromCache(cachePath string) ([]SonarrSeries, bool) {
	cacheData, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, false
	}
	var series []SonarrSeries
	if err := json.Unmarshal(cacheData, &series); err != nil {
		return nil, false
	}
	return series, true
}

func getSonarrSettingsFromConfig() (struct {
	URL    string
	APIKey string
}, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return struct {
			URL    string
			APIKey string
		}{}, err
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		return struct {
			URL    string
			APIKey string
		}{}, err
	}
	return struct {
		URL    string
		APIKey string
	}{
		URL:    allSettings.Sonarr.URL,
		APIKey: allSettings.Sonarr.APIKey,
	}, nil
}

func fetchAndFilterSonarrSeries(apiBase, apiKey string) ([]SonarrSeries, error) {
	req, err := http.NewRequest("GET", apiBase+"/api/v3/series", nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, apiKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error fetching series: %w", err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[HandleSonarrSeries] Raw response body: %s\n", string(body))
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Sonarr API error: %d", resp.StatusCode)
	}
	var allSeries []map[string]interface{}
	if err := json.Unmarshal(body, &allSeries); err != nil {
		return nil, fmt.Errorf("Failed to decode Sonarr response: %w", err)
	}
	return filterDownloadedSonarrSeries(allSeries), nil
}

func filterDownloadedSonarrSeries(allSeries []map[string]interface{}) []SonarrSeries {
	series := make([]SonarrSeries, 0)
	for _, s := range allSeries {
		stats, ok := s["statistics"].(map[string]interface{})
		if !ok {
			continue
		}
		episodeFileCount, ok := stats["episodeFileCount"].(float64)
		if !ok || episodeFileCount < 1 {
			continue
		}
		id, ok := s["id"].(float64)
		if !ok {
			continue
		}
		title, _ := s["title"].(string)
		year, _ := s["year"].(float64)
		path, _ := s["path"].(string)
		series = append(series, SonarrSeries{
			ID:    int(id),
			Title: title,
			Year:  int(year),
			Path:  path,
		})
	}
	return series
}

// --- Sonarr Series Sync ---
func SyncSonarrSeriesAndMediaCover() error {
	var syncErr error
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		fmt.Println("[SyncSonarr] Sonarr settings not found")
		return fmt.Errorf("Sonarr settings not found: %w", err)
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		fmt.Println("[SyncSonarr] Invalid Sonarr settings")
		return fmt.Errorf("Invalid Sonarr settings: %w", err)
	}
	settings := allSettings.Sonarr
	cachePath := SeriesCachePath
	req, err := http.NewRequest("GET", settings.URL+"/api/v3/series", nil)
	if err != nil {
		fmt.Println("[SyncSonarr] Error creating request:", err)
		return fmt.Errorf("Error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, settings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[SyncSonarr] Error fetching series:", err)
		return fmt.Errorf("Error fetching series: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("[SyncSonarr] Sonarr API error:", resp.StatusCode)
		return fmt.Errorf("Sonarr API error: %d", resp.StatusCode)
	}
	var allSeries []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allSeries); err != nil {
		fmt.Println("[SyncSonarr] Failed to decode Sonarr response:", err)
		return fmt.Errorf("Failed to decode Sonarr response: %w", err)
	}
	// Filter only series with downloaded episodes
	series := make([]map[string]interface{}, 0)
	for _, s := range allSeries {
		stats, ok := s["statistics"].(map[string]interface{})
		if !ok {
			continue
		}
		episodeFileCount, ok := stats["episodeFileCount"].(float64)
		if !ok || episodeFileCount < 1 {
			continue
		}
		series = append(series, s)
	}
	cacheData, _ := json.MarshalIndent(series, "", "  ")
	_ = os.WriteFile(cachePath, cacheData, 0644)
	fmt.Println("[SyncSonarr] Synced", len(series), "downloaded series to cache.")
	return syncErr
}

// Handler to list existing extras for a movie path
func existingExtrasHandler(c *gin.Context) {
	moviePath := c.Query("moviePath")
	if moviePath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "moviePath required"})
		return
	}
	// Scan subfolders for .mp4 files and their metadata
	var existing []map[string]interface{}
	entries, err := os.ReadDir(moviePath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"existing": []map[string]interface{}{}})
		return
	}
	// Track duplicate index for each type/title
	dupCount := make(map[string]int)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		subdir := moviePath + "/" + entry.Name()
		files, _ := os.ReadDir(subdir)
		for _, f := range files {
			if !f.IsDir() && strings.HasSuffix(f.Name(), ".mp4") {
				metaFile := subdir + "/" + strings.TrimSuffix(f.Name(), ".mp4") + ".mp4.json"
				var meta struct {
					Type      string `json:"type"`
					Title     string `json:"title"`
					YouTubeID string `json:"youtube_id"`
				}
				if metaBytes, err := os.ReadFile(metaFile); err == nil {
					_ = json.Unmarshal(metaBytes, &meta)
				}
				key := entry.Name() + "|" + meta.Title
				dupCount[key]++
				existing = append(existing, map[string]interface{}{
					"type":       entry.Name(),
					"title":      meta.Title,
					"youtube_id": meta.YouTubeID,
					"_dupIndex":  dupCount[key],
				})
			}
		}
	}
	c.JSON(http.StatusOK, gin.H{"existing": existing})
}

// Handler to fetch movies from Radarr
func getRadarrMoviesHandler(c *gin.Context) {
	// Serve movies from cache (only movies with downloaded posters)
	cachePath := MoviesCachePath
	fmt.Println("[getRadarrMoviesHandler] cachePath:", cachePath)
	cacheData, err := os.ReadFile(cachePath)
	if err != nil {
		fmt.Println("[getRadarrMoviesHandler] Error reading cache:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Movie cache not found"})
		return
	}
	var movies []map[string]interface{}
	if err := json.Unmarshal(cacheData, &movies); err != nil {
		fmt.Println("[getRadarrMoviesHandler] Error decoding cache:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode movie cache"})
		return
	}
	fmt.Printf("[getRadarrMoviesHandler] Loaded %d movies from cache\n", len(movies))
	c.JSON(http.StatusOK, gin.H{"movies": movies})
}

// Handler to get Radarr settings
func getRadarrSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	var allSettings struct {
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": allSettings.Radarr.URL, "apiKey": allSettings.Radarr.APIKey})
}

// Handler to save Radarr settings
func saveRadarrSettingsHandler(c *gin.Context) {
	var req struct {
		URL    string `yaml:"url"`
		APIKey string `yaml:"apiKey"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	// Read existing settings
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	data, _ := os.ReadFile(ConfigPath)
	_ = yaml.Unmarshal(data, &allSettings)
	allSettings.Radarr.URL = req.URL
	allSettings.Radarr.APIKey = req.APIKey
	out, _ := yaml.Marshal(allSettings)
	err := os.WriteFile(ConfigPath, out, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// Handler for Plex items
func plexItemsHandler(c *gin.Context) {
	items, err := FetchPlexLibrary()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func searchExtrasHandler(c *gin.Context) {
	mediaType := c.Query("mediaType")
	idStr := c.Query("id")
	var id int
	fmt.Sscanf(idStr, "%d", &id)
	results, _ := SearchExtras(mediaType, id)
	c.JSON(http.StatusOK, gin.H{"extras": results})
}

func downloadExtraHandler(c *gin.Context) {
	var req struct {
		MoviePath  string `json:"moviePath"`
		ExtraType  string `json:"extraType"`
		ExtraTitle string `json:"extraTitle"`
		URL        string `json:"url"`
	}
	if err := c.BindJSON(&req); err != nil {
		fmt.Printf("[downloadExtraHandler] Invalid request: %v\n", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	fmt.Printf("[downloadExtraHandler] Download request: moviePath=%s, extraType=%s, extraTitle=%s, url=%s\n", req.MoviePath, req.ExtraType, req.ExtraTitle, req.URL)
	meta, err := DownloadYouTubeExtra(req.MoviePath, req.ExtraType, req.ExtraTitle, req.URL)
	if err != nil {
		fmt.Printf("[downloadExtraHandler] Download error: %v\n", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "downloaded", "meta": meta})
}

// SyncRadarrMoviesAndMediaCover syncs Radarr movie list and MediaCover folder
func SyncRadarrMoviesAndMediaCover() error {
	settings, err := loadRadarrSettings()
	if err != nil {
		fmt.Println("[Sync] Radarr settings not found or invalid:", err)
		return err
	}

	movies, err := fetchDownloadedRadarrMovies(settings)
	if err != nil {
		fmt.Println("[Sync] Error fetching movies:", err)
		return err
	}

	if err := cacheMovies(movies, MoviesCachePath); err != nil {
		fmt.Println("[Sync] Error caching movies:", err)
		return err
	}
	fmt.Println("[Sync] Synced", len(movies), "downloaded movies to cache.")

	downloadedMovies, err := downloadMovieImages(settings, movies)
	if err != nil {
		fmt.Println("[Sync] Error downloading images:", err)
	}

	if err := atomicCacheMovies(downloadedMovies, "/var/lib/extrazarr/movies_cache.json"); err != nil {
		fmt.Println("[Sync] Error updating cache with posters:", err)
		return err
	}
	fmt.Println("[Sync] Cached", len(downloadedMovies), "movies with posters.")

	return err
}

func loadRadarrSettings() (struct {
	URL    string
	APIKey string
}, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return struct {
			URL    string
			APIKey string
		}{}, fmt.Errorf("Radarr settings not found: %w", err)
	}
	var allSettings struct {
		Radarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"radarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		return struct {
			URL    string
			APIKey string
		}{}, fmt.Errorf("Invalid Radarr settings: %w", err)
	}
	return struct {
		URL    string
		APIKey string
	}{
		URL:    allSettings.Radarr.URL,
		APIKey: allSettings.Radarr.APIKey,
	}, nil
}

func fetchDownloadedRadarrMovies(settings struct{ URL, APIKey string }) ([]map[string]interface{}, error) {
	req, err := http.NewRequest("GET", settings.URL+"/api/v3/movie", nil)
	if err != nil {
		return nil, fmt.Errorf("Error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, settings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("Error fetching movies: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Radarr API error: %d", resp.StatusCode)
	}
	var allMovies []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allMovies); err != nil {
		return nil, fmt.Errorf("Failed to decode Radarr response: %w", err)
	}
	movies := make([]map[string]interface{}, 0)
	for _, m := range allMovies {
		if hasFile, ok := m["hasFile"].(bool); ok && hasFile {
			movies = append(movies, m)
		}
	}
	return movies, nil
}

func cacheMovies(movies []map[string]interface{}, cachePath string) error {
	cacheData, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(cachePath, cacheData, 0644)
}

func downloadMovieImages(settings struct{ URL, APIKey string }, movies []map[string]interface{}) ([]map[string]interface{}, error) {
	client := &http.Client{}
	downloadedMovies := make([]map[string]interface{}, 0)
	for _, movie := range movies {
		id, ok := movie["id"].(float64)
		if !ok {
			continue
		}
		idStr := fmt.Sprintf("%d", int(id))
		posterUrl := fmt.Sprintf("%s/MediaCover/%s/poster-500.jpg", settings.URL, idStr)
		fanartUrl := fmt.Sprintf("%s/MediaCover/%s/fanart-1280.jpg", settings.URL, idStr)
		posterPath := fmt.Sprintf("/var/lib/extrazarr/MediaCover/%s/poster-500.jpg", idStr)
		fanartPath := fmt.Sprintf("/var/lib/extrazarr/MediaCover/%s/fanart-1280.jpg", idStr)

		os.MkdirAll(fmt.Sprintf("/var/lib/extrazarr/MediaCover/%s", idStr), 0755)

		if downloadImage(client, posterUrl, posterPath) {
			downloadedMovies = append(downloadedMovies, movie)
		}
		downloadImage(client, fanartUrl, fanartPath)
	}
	return downloadedMovies, nil
}

func downloadImage(client *http.Client, url, path string) bool {
	resp, err := client.Get(url)
	if err != nil {
		fmt.Println("[Sync] Failed to download image:", url, err)
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("[Sync] Image not found:", url)
		return false
	}
	out, err := os.Create(path)
	if err != nil {
		fmt.Println("[Sync] Error creating file for image:", path, err)
		return false
	}
	defer out.Close()
	if _, err = io.Copy(out, resp.Body); err != nil {
		fmt.Println("[Sync] Error saving image:", path, err)
		return false
	}
	return true
}

func atomicCacheMovies(movies []map[string]interface{}, cachePath string) error {
	tmpCachePath := cachePath + ".tmp"
	cacheData, err := json.MarshalIndent(movies, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(tmpCachePath, cacheData, 0644); err != nil {
		return err
	}
	return os.Rename(tmpCachePath, cachePath)
}
