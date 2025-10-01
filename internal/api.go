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
}

// Handler for /api/radarr/poster/:movieId
func HandleRadarrPoster(c *gin.Context) {
	movieId := c.Param("movieId")
	// Load Radarr settings
	data, err := os.ReadFile("settings.json")
	var allSettings struct {
		Radarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"radarr"`
	}
	if err := json.Unmarshal(data, &allSettings); err != nil {
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
	localPath := "/var/lib/extrazarr/MediaCover/" + movieId + "/poster-500.jpg"
	if _, err := os.Stat(localPath); err == nil {
		c.File(localPath)
		return
	}
	// Fallback to Radarr API
	posterUrl := apiBase + "/MediaCover/" + movieId + "/poster-500.jpg"
	req, err := http.NewRequest("GET", posterUrl, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating poster request")
		return
	}
	req.Header.Set("X-Api-Key", radarrSettings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch poster image from Radarr")
		return
	}
	defer resp.Body.Close()
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}

// Handler for /api/radarr/banner/:movieId (fanart)
func HandleRadarrBanner(c *gin.Context) {
	movieId := c.Param("movieId")
	// Load Radarr settings
	data, err := os.ReadFile("settings.json")
	var allSettings struct {
		Radarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"radarr"`
	}
	if err := json.Unmarshal(data, &allSettings); err != nil {
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
	localPath := "/var/lib/extrazarr/MediaCover/" + movieId + "/fanart-1280.jpg"
	if _, err := os.Stat(localPath); err == nil {
		c.File(localPath)
		return
	}
	// Fallback to Radarr API
	bannerUrl := apiBase + "/MediaCover/" + movieId + "/fanart-1280.jpg"
	req, err := http.NewRequest("GET", bannerUrl, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating banner request")
		return
	}
	req.Header.Set("X-Api-Key", radarrSettings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch banner image from Radarr")
		return
	}
	defer resp.Body.Close()
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, resp.Body)
}

// Handler for /api/sonarr/banner/:seriesId
func HandleSonarrBanner(c *gin.Context) {
	seriesId := c.Param("seriesId")
	// Load Sonarr settings
	data, err := os.ReadFile("settings.json")
	var allSettings struct {
		Sonarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"sonarr"`
	}
	if err := json.Unmarshal(data, &allSettings); err != nil {
		c.String(http.StatusInternalServerError, "Invalid Sonarr settings")
		return
	}
	sonarrSettings := allSettings.Sonarr
	// Remove trailing slash from URL if present
	apiBase := sonarrSettings.URL
	if strings.HasSuffix(apiBase, "/") {
		apiBase = strings.TrimRight(apiBase, "/")
	}
	// Fetch series info from Sonarr to get banner path
	req, err := http.NewRequest("GET", apiBase+"/api/v3/series/"+seriesId, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating request")
		return
	}
	req.Header.Set("X-Api-Key", sonarrSettings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch series info from Sonarr")
		return
	}
	defer resp.Body.Close()
	var series map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		c.String(http.StatusInternalServerError, "Failed to decode Sonarr response")
		return
	}
	// Find banner path in images array
	images, ok := series["images"].([]interface{})
	var bannerUrl string
	if ok {
		for _, img := range images {
			m, ok := img.(map[string]interface{})
			if ok && m["coverType"] == "banner" {
				if remoteUrl, ok := m["remoteUrl"].(string); ok && remoteUrl != "" {
					bannerUrl = remoteUrl
					break
				}
				if url, ok := m["url"].(string); ok && url != "" {
					bannerUrl = apiBase + url
					break
				}
			}
		}
	}
	if bannerUrl == "" {
		c.String(http.StatusNotFound, "No banner found for series")
		return
	}
	// Proxy the banner image
	bannerReq, err := http.NewRequest("GET", bannerUrl, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating banner request")
		return
	}
	// If banner is local, add API key
	if strings.HasPrefix(bannerUrl, apiBase) {
		bannerReq.Header.Set("X-Api-Key", sonarrSettings.APIKey)
	}
	bannerResp, err := client.Do(bannerReq)
	if err != nil || bannerResp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch banner image")
		return
	}
	defer bannerResp.Body.Close()
	c.Header("Content-Type", bannerResp.Header.Get("Content-Type"))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, bannerResp.Body)
}

// Handler for /api/sonarr/poster/:seriesId
func HandleSonarrPoster(c *gin.Context) {
	seriesId := c.Param("seriesId")
	// Load Sonarr settings
	data, err := os.ReadFile("settings.json")
	var allSettings struct {
		Sonarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"sonarr"`
	}
	if err := json.Unmarshal(data, &allSettings); err != nil {
		c.String(http.StatusInternalServerError, "Invalid Sonarr settings")
		return
	}
	sonarrSettings := allSettings.Sonarr
	// Remove trailing slash from URL if present
	apiBase := sonarrSettings.URL
	if strings.HasSuffix(apiBase, "/") {
		apiBase = strings.TrimRight(apiBase, "/")
	}
	// Fetch series info from Sonarr to get poster path
	req, err := http.NewRequest("GET", apiBase+"/api/v3/series/"+seriesId, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating request")
		return
	}
	req.Header.Set("X-Api-Key", sonarrSettings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil || resp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch series info from Sonarr")
		return
	}
	defer resp.Body.Close()
	var series map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&series); err != nil {
		c.String(http.StatusInternalServerError, "Failed to decode Sonarr response")
		return
	}
	// Find poster path in images array
	images, ok := series["images"].([]interface{})
	var posterUrl string
	if ok {
		for _, img := range images {
			m, ok := img.(map[string]interface{})
			if ok && m["coverType"] == "poster" {
				if remoteUrl, ok := m["remoteUrl"].(string); ok && remoteUrl != "" {
					posterUrl = remoteUrl
					break
				}
				if url, ok := m["url"].(string); ok && url != "" {
					posterUrl = apiBase + url
					break
				}
			}
		}
	}
	if posterUrl == "" {
		c.String(http.StatusNotFound, "No poster found for series")
		return
	}
	// Proxy the poster image
	posterReq, err := http.NewRequest("GET", posterUrl, nil)
	if err != nil {
		c.String(http.StatusInternalServerError, "Error creating poster request")
		return
	}
	// If poster is local, add API key
	if strings.HasPrefix(posterUrl, apiBase) {
		posterReq.Header.Set("X-Api-Key", sonarrSettings.APIKey)
	}
	posterResp, err := client.Do(posterReq)
	if err != nil || posterResp.StatusCode != 200 {
		c.String(http.StatusBadGateway, "Failed to fetch poster image")
		return
	}
	defer posterResp.Body.Close()
	c.Header("Content-Type", posterResp.Header.Get("Content-Type"))
	c.Status(http.StatusOK)
	io.Copy(c.Writer, posterResp.Body)
}

// Handler to get Sonarr settings
func getSonarrSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile("settings.json")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"sonarr"`
	}
	if err := json.Unmarshal(data, &allSettings); err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": allSettings.Sonarr.URL, "apiKey": allSettings.Sonarr.APIKey})
}

// Handler to save Sonarr settings
func saveSonarrSettingsHandler(c *gin.Context) {
	var req struct {
		URL    string `json:"url"`
		APIKey string `json:"apiKey"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	// Read existing settings
	var allSettings struct {
		Sonarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"sonarr"`
		Radarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"radarr"`
	}
	data, _ := os.ReadFile("settings.json")
	_ = json.Unmarshal(data, &allSettings)
	allSettings.Sonarr.URL = req.URL
	allSettings.Sonarr.APIKey = req.APIKey
	out, _ := json.MarshalIndent(allSettings, "", "  ")
	err := os.WriteFile("settings.json", out, 0644)
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
	// Serve series from cache (only series with downloaded posters)
	cachePath := "/var/lib/extrazarr/series_cache.json"
	cacheData, err := os.ReadFile(cachePath)
	if err == nil {
		var series []SonarrSeries
		if err := json.Unmarshal(cacheData, &series); err == nil {
			c.JSON(http.StatusOK, gin.H{"series": series})
			return
		}
	}

	// Load Sonarr settings
	data, err := os.ReadFile("settings.json")
	if err != nil {
		fmt.Println("[HandleSonarrSeries] Sonarr settings not found")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sonarr settings not found"})
		return
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"sonarr"`
	}
	if err := json.Unmarshal(data, &allSettings); err != nil {
		fmt.Println("[HandleSonarrSeries] Invalid Sonarr settings")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid Sonarr settings"})
		return
	}
	sonarrSettings := allSettings.Sonarr

	// Fetch series from Sonarr
	// Remove trailing slash from URL if present
	apiBase := sonarrSettings.URL
	if strings.HasSuffix(apiBase, "/") {
		apiBase = strings.TrimRight(apiBase, "/")
	}
	req, err := http.NewRequest("GET", apiBase+"/api/v3/series", nil)
	if err != nil {
		fmt.Println("[HandleSonarrSeries] Error creating request:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error creating request"})
		return
	}
	req.Header.Set("X-Api-Key", sonarrSettings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[HandleSonarrSeries] Error fetching series:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching series", "details": err.Error()})
		return
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("[HandleSonarrSeries] Raw response body: %s\n", string(body))
	if resp.StatusCode != 200 {
		fmt.Printf("[HandleSonarrSeries] Sonarr API error: %d\n", resp.StatusCode)
		return
	}
	var allSeries []map[string]interface{}
	if err := json.Unmarshal(body, &allSeries); err != nil {
		fmt.Println("[HandleSonarrSeries] Failed to decode Sonarr response:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to decode Sonarr response", "details": err.Error(), "body": string(body)})
		return
	}
	// Filter only series with downloaded episodes (statistics.episodeFileCount > 0)
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
	// Save series to cache file
	cacheData, _ = json.MarshalIndent(series, "", "  ")
	_ = os.WriteFile(cachePath, cacheData, 0644)
	c.JSON(http.StatusOK, gin.H{"series": series})
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
	cachePath := "/var/lib/extrazarr/movies_cache.json"
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
	data, err := os.ReadFile("settings.json")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	var allSettings struct {
		Radarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"radarr"`
	}
	if err := json.Unmarshal(data, &allSettings); err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": allSettings.Radarr.URL, "apiKey": allSettings.Radarr.APIKey})
}

// Handler to save Radarr settings
func saveRadarrSettingsHandler(c *gin.Context) {
	var req struct {
		URL    string `json:"url"`
		APIKey string `json:"apiKey"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	// Read existing settings
	var allSettings struct {
		Sonarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"sonarr"`
		Radarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"radarr"`
	}
	data, _ := os.ReadFile("settings.json")
	_ = json.Unmarshal(data, &allSettings)
	allSettings.Radarr.URL = req.URL
	allSettings.Radarr.APIKey = req.APIKey
	out, _ := json.MarshalIndent(allSettings, "", "  ")
	err := os.WriteFile("settings.json", out, 0644)
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
	movie := c.Query("movie")
	results, _ := SearchExtras(movie)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
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
func SyncRadarrMoviesAndMediaCover() {
	// Load Radarr settings
	data, err := os.ReadFile("settings.json")
	if err != nil {
		fmt.Println("[Sync] Radarr settings not found")
		return
	}
	var allSettings struct {
		Radarr struct {
			URL    string `json:"url"`
			APIKey string `json:"apiKey"`
		} `json:"radarr"`
	}
	if err := json.Unmarshal(data, &allSettings); err != nil {
		fmt.Println("[Sync] Invalid Radarr settings")
		return
	}
	settings := allSettings.Radarr
	// Always fetch movies from Radarr and refresh cache
	cachePath := "/var/lib/extrazarr/movies_cache.json"
	req, err := http.NewRequest("GET", settings.URL+"/api/v3/movie", nil)
	if err != nil {
		fmt.Println("[Sync] Error creating request:", err)
		return
	}
	req.Header.Set("X-Api-Key", settings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[Sync] Error fetching movies:", err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("[Sync] Radarr API error:", resp.StatusCode)
		return
	}
	var allMovies []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allMovies); err != nil {
		fmt.Println("[Sync] Failed to decode Radarr response:", err)
		return
	}
	// Filter only downloaded movies (hasFile == true)
	movies := make([]map[string]interface{}, 0)
	for _, m := range allMovies {
		if hasFile, ok := m["hasFile"].(bool); ok && hasFile {
			movies = append(movies, m)
		}
	}
	// Save movies to cache file
	cacheData, _ := json.MarshalIndent(movies, "", "  ")
	_ = os.WriteFile(cachePath, cacheData, 0644)
	fmt.Println("[Sync] Synced", len(movies), "downloaded movies to cache.")

	// Download poster images from Radarr and cache only movies with posters
	client = &http.Client{}
	downloadedMovies := make([]map[string]interface{}, 0)
	for _, movie := range movies {
		hasFile, ok := movie["hasFile"].(bool)
		if !ok || !hasFile {
			continue
		}
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

		// Download poster
		resp, err := client.Get(posterUrl)
		if err != nil {
			fmt.Println("[Sync] Failed to download poster for movie", idStr, err)
		} else {
			defer resp.Body.Close()
			if resp.StatusCode == 200 {
				out, err := os.Create(posterPath)
				if err == nil {
					_, err = io.Copy(out, resp.Body)
					out.Close()
					if err == nil {
						downloadedMovies = append(downloadedMovies, movie)
					} else {
						fmt.Println("[Sync] Error saving poster for movie", idStr, err)
					}
				} else {
					fmt.Println("[Sync] Error creating file for poster", posterPath, err)
				}
			} else {
				fmt.Println("[Sync] Poster not found for movie", idStr)
			}
		}

		// Download fanart for background
		respFanart, err := client.Get(fanartUrl)
		if err != nil {
			fmt.Println("[Sync] Failed to download fanart for movie", idStr, err)
			continue
		}
		defer respFanart.Body.Close()
		if respFanart.StatusCode == 200 {
			out, err := os.Create(fanartPath)
			if err == nil {
				_, err = io.Copy(out, respFanart.Body)
				out.Close()
				if err != nil {
					fmt.Println("[Sync] Error saving fanart for movie", idStr, err)
				}
			} else {
				fmt.Println("[Sync] Error creating file for fanart", fanartPath, err)
			}
		} else {
			fmt.Println("[Sync] Fanart not found for movie", idStr)
		}
	}

	// Save only movies with downloaded posters to cache
	cachePath = "/var/lib/extrazarr/movies_cache.json"
	tmpCachePath := cachePath + ".tmp"
	cacheData, _ = json.MarshalIndent(downloadedMovies, "", "  ")
	// Write to a temporary file first
	if err := os.WriteFile(tmpCachePath, cacheData, 0644); err != nil {
		fmt.Println("[Sync] Error writing temp cache file:", err)
	} else {
		// Atomically replace the cache file only after successful write
		if err := os.Rename(tmpCachePath, cachePath); err != nil {
			fmt.Println("[Sync] Error replacing cache file:", err)
		} else {
			fmt.Println("[Sync] Cached", len(downloadedMovies), "movies with posters.")
		}
	}
}
