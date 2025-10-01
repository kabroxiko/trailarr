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

func getRadarrPosterHandler(c *gin.Context) {
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

func getRadarrBannerHandler(c *gin.Context) {
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
}

func getRadarrMoviesHandler(c *gin.Context) {
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

func saveRadarrSettingsHandler(c *gin.Context) {
	var req struct {
		URL    string `yaml:"url"`
		APIKey string `yaml:"apiKey"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
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
