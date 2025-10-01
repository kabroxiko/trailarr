package internal

import (
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
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
