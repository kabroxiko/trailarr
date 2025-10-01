package internal

import (
	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
	"io"
	"net/http"
	"os"
	"strings"
)

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
	// Handler functions will be moved from api.go

	// Handler functions will be moved from api.go
}
