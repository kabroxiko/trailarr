package internal

import (
	"fmt"
	"net/http"
	"os"
	"time"

	"encoding/json"

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
	// Read existing settings as map[string]interface{} to preserve all keys
	data, _ := os.ReadFile(ConfigPath)
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	// Update only general section
	if config["general"] == nil {
		config["general"] = map[string]interface{}{}
	}
	general := config["general"].(map[string]interface{})
	general["tmdbKey"] = req.TMDBApiKey
	config["general"] = general
	out, _ := yaml.Marshal(config)
	err := os.WriteFile(ConfigPath, out, 0644)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// Fetch root folders from Radarr or Sonarr API
func FetchRootFolders(apiURL, apiKey string) ([]map[string]interface{}, error) {
	req, err := http.NewRequest("GET", apiURL+"/api/v3/rootfolder", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Api-Key", apiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	var folders []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&folders); err != nil {
		return nil, err
	}
	// Only return root folder paths
	var rootFolderPaths []map[string]interface{}
	for _, folder := range folders {
		if path, ok := folder["path"].(string); ok {
			rootFolderPaths = append(rootFolderPaths, map[string]interface{}{"path": path})
		}
	}
	return rootFolderPaths, nil
}
