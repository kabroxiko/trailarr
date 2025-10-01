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

// Handler for /api/sonarr/banner/:seriesId
func getSonarrBannerHandler(c *gin.Context) {
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
func getSonarrPosterHandler(c *gin.Context) {
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

// Handler for /api/sonarr/series
func getSonarrSeriesHandler(c *gin.Context) {
	cachePath := SeriesCachePath
	if series, ok := loadSonarrSeriesFromCache(cachePath); ok {
		c.JSON(http.StatusOK, gin.H{"series": series})
		return
	}

	sonarrSettings, err := getSonarrSettingsFromConfig()
	if err != nil {
		fmt.Println("[getSonarrSeriesHandler] Sonarr settings not found or invalid:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Sonarr settings not found"})
		return
	}

	apiBase := trimTrailingSlash(sonarrSettings.URL)
	series, err := fetchAndFilterSonarrSeries(apiBase, sonarrSettings.APIKey)
	if err != nil {
		fmt.Println("[getSonarrSeriesHandler] Error fetching or decoding series:", err)
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
	fmt.Printf("[getSonarrSeriesHandler] Raw response body: %s\n", string(body))
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
		overview, _ := s["overview"].(string)
		series = append(series, SonarrSeries{
			ID:       int(id),
			Title:    title,
			Year:     int(year),
			Path:     path,
			Overview: overview,
		})
	}
	return series
}
