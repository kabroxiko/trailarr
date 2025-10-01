package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	r.POST("/api/settings/radarr", saveRadarrSettingsHandler)
	r.GET("/api/settings/radarr", getRadarrSettingsHandler)
	r.GET("/api/extras/search", searchExtrasHandler)
	r.POST("/api/extras/download", downloadExtraHandler)
	r.GET("/api/plex", plexItemsHandler)
}

// Handler to get Radarr settings
func getRadarrSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile("radarr.json")
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	var settings struct {
		URL    string `json:"url"`
		APIKey string `json:"apiKey"`
	}
	if err := json.Unmarshal(data, &settings); err != nil {
		c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
		return
	}
	c.JSON(http.StatusOK, gin.H{"url": settings.URL, "apiKey": settings.APIKey})
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
	// Save to a config file (radarr.json)
	data := []byte(fmt.Sprintf(`{"url": "%s", "apiKey": "%s"}`, req.URL, req.APIKey))
	err := os.WriteFile("radarr.json", data, 0644)
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
		URL string `json:"url"`
	}
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
		return
	}
	_ = DownloadExtra(req.URL)
	c.JSON(http.StatusOK, gin.H{"status": "downloading"})
}
