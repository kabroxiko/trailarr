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

var Timings map[string]int

// SearchExtrasConfig holds config for searching movie/series extras
type SearchExtrasConfig struct {
	SearchMoviesExtras bool `yaml:"searchMoviesExtras" json:"searchMoviesExtras"`
	SearchSeriesExtras bool `yaml:"searchSeriesExtras" json:"searchSeriesExtras"`
	AutoDownloadExtras bool `yaml:"autoDownloadExtras" json:"autoDownloadExtras"`
}

// ExtraTypesConfig holds config for enabling/disabling specific extra types
type ExtraTypesConfig struct {
	Trailers        bool `yaml:"trailers" json:"trailers"`
	Scenes          bool `yaml:"scenes" json:"scenes"`
	BehindTheScenes bool `yaml:"behindTheScenes" json:"behindTheScenes"`
	Interviews      bool `yaml:"interviews" json:"interviews"`
	Featurettes     bool `yaml:"featurettes" json:"featurettes"`
	DeletedScenes   bool `yaml:"deletedScenes" json:"deletedScenes"`
	Other           bool `yaml:"other" json:"other"`
}

// GetExtraTypesConfig loads extra types config from config.yml
func GetExtraTypesConfig() (ExtraTypesConfig, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return ExtraTypesConfig{Trailers: true, Scenes: true, BehindTheScenes: true, Interviews: true, Featurettes: true, DeletedScenes: true, Other: true}, err
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return ExtraTypesConfig{Trailers: true, Scenes: true, BehindTheScenes: true, Interviews: true, Featurettes: true, DeletedScenes: true, Other: true}, err
	}
	sec, ok := config["extraTypes"].(map[string]interface{})
	cfg := ExtraTypesConfig{}
	if !ok {
		// Default: all enabled
		return ExtraTypesConfig{Trailers: true, Scenes: true, BehindTheScenes: true, Interviews: true, Featurettes: true, DeletedScenes: true, Other: true}, nil
	}
	if v, ok := sec["trailers"].(bool); ok {
		cfg.Trailers = v
	} else {
		cfg.Trailers = true
	}
	if v, ok := sec["scenes"].(bool); ok {
		cfg.Scenes = v
	} else {
		cfg.Scenes = true
	}
	if v, ok := sec["behindTheScenes"].(bool); ok {
		cfg.BehindTheScenes = v
	} else {
		cfg.BehindTheScenes = true
	}
	if v, ok := sec["interviews"].(bool); ok {
		cfg.Interviews = v
	} else {
		cfg.Interviews = true
	}
	if v, ok := sec["featurettes"].(bool); ok {
		cfg.Featurettes = v
	} else {
		cfg.Featurettes = true
	}
	if v, ok := sec["deletedScenes"].(bool); ok {
		cfg.DeletedScenes = v
	} else {
		cfg.DeletedScenes = true
	}
	if v, ok := sec["other"].(bool); ok {
		cfg.Other = v
	} else {
		cfg.Other = true
	}
	return cfg, nil
}

// SaveExtraTypesConfig saves extra types config to config.yml
func SaveExtraTypesConfig(cfg ExtraTypesConfig) error {
	data, _ := os.ReadFile(ConfigPath)
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	config["extraTypes"] = map[string]interface{}{
		"trailers":        cfg.Trailers,
		"scenes":          cfg.Scenes,
		"behindTheScenes": cfg.BehindTheScenes,
		"interviews":      cfg.Interviews,
		"featurettes":     cfg.Featurettes,
		"deletedScenes":   cfg.DeletedScenes,
		"other":           cfg.Other,
	}
	out, _ := yaml.Marshal(config)
	return os.WriteFile(ConfigPath, out, 0644)
}

// Handler to get extra types config
func GetExtraTypesConfigHandler(c *gin.Context) {
	cfg, err := GetExtraTypesConfig()
	if err != nil {
		c.JSON(http.StatusOK, cfg)
		return
	}
	c.JSON(http.StatusOK, cfg)
}

// Handler to save extra types config
func SaveExtraTypesConfigHandler(c *gin.Context) {
	var req ExtraTypesConfig
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	if err := SaveExtraTypesConfig(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// GetAutoDownloadExtras reads the autoDownloadExtras flag from config.yml (general section)
func GetAutoDownloadExtras() bool {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return true // default enabled
	}
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	if general, ok := config["general"].(map[string]interface{}); ok {
		if v, ok := general["autoDownloadExtras"].(bool); ok {
			return v
		}
	}
	return true
}

// GetSearchExtrasConfig loads search extras config from config.yml
func GetSearchExtrasConfig() (SearchExtrasConfig, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return SearchExtrasConfig{}, err
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return SearchExtrasConfig{}, err
	}
	sec, ok := config["searchExtras"].(map[string]interface{})
	if !ok {
		// Default: both enabled, autoDownload enabled
		return SearchExtrasConfig{SearchMoviesExtras: true, SearchSeriesExtras: true, AutoDownloadExtras: true}, nil
	}
	cfg := SearchExtrasConfig{}
	if v, ok := sec["searchMoviesExtras"].(bool); ok {
		cfg.SearchMoviesExtras = v
	} else {
		cfg.SearchMoviesExtras = true
	}
	if v, ok := sec["searchSeriesExtras"].(bool); ok {
		cfg.SearchSeriesExtras = v
	} else {
		cfg.SearchSeriesExtras = true
	}
	if v, ok := sec["autoDownloadExtras"].(bool); ok {
		cfg.AutoDownloadExtras = v
	} else {
		cfg.AutoDownloadExtras = true
	}
	return cfg, nil
}

// SaveSearchExtrasConfig saves search extras config to config.yml
func SaveSearchExtrasConfig(cfg SearchExtrasConfig) error {
	data, _ := os.ReadFile(ConfigPath)
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	config["searchExtras"] = map[string]interface{}{
		"searchMoviesExtras": cfg.SearchMoviesExtras,
		"searchSeriesExtras": cfg.SearchSeriesExtras,
		"autoDownloadExtras": cfg.AutoDownloadExtras,
	}
	out, _ := yaml.Marshal(config)
	return os.WriteFile(ConfigPath, out, 0644)
}

// Handler to get search extras config
func GetSearchExtrasConfigHandler(c *gin.Context) {
	cfg, err := GetSearchExtrasConfig()
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"searchMoviesExtras": true, "searchSeriesExtras": true})
		return
	}
	c.JSON(http.StatusOK, gin.H{"searchMoviesExtras": cfg.SearchMoviesExtras, "searchSeriesExtras": cfg.SearchSeriesExtras})
}

// Handler to save search extras config
func SaveSearchExtrasConfigHandler(c *gin.Context) {
	var req SearchExtrasConfig
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
		return
	}
	if err := SaveSearchExtrasConfig(req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}

// EnsureSyncTimingsConfig creates config.yml with sync timings if not present, or loads timings if present
func EnsureSyncTimingsConfig() (map[string]int, error) {
	defaultTimings := map[string]int{
		"radarr": 15,
		"sonarr": 15,
	}
	// Check if config file exists
	if _, err := os.Stat(ConfigPath); os.IsNotExist(err) {
		// Create config with only syncTimings
		cfg := map[string]interface{}{"syncTimings": defaultTimings}
		out, err := yaml.Marshal(cfg)
		if err != nil {
			return defaultTimings, err
		}
		// Ensure parent dir exists
		if err := os.MkdirAll(TrailarrRoot+"/config", 0775); err != nil {
			return defaultTimings, err
		}
		if err := os.WriteFile(ConfigPath, out, 0644); err != nil {
			return defaultTimings, err
		}
		return defaultTimings, nil
	}
	// Load config
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return defaultTimings, err
	}
	var cfg map[string]interface{}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return defaultTimings, err
	}
	timings, ok := cfg["syncTimings"].(map[string]interface{})
	if !ok || len(timings) == 0 {
		// Add syncTimings without touching other config
		cfg["syncTimings"] = defaultTimings
		out, err := yaml.Marshal(cfg)
		if err == nil {
			_ = os.WriteFile(ConfigPath, out, 0644)
		}
		return defaultTimings, nil
	}
	// Convert loaded timings to map[string]int (robust for all numeric types)
	result := map[string]int{}
	for k, v := range timings {
		switch val := v.(type) {
		case int:
			result[k] = val
		case int64:
			result[k] = int(val)
		case float64:
			result[k] = int(val)
		case uint64:
			result[k] = int(val)
		case uint:
			result[k] = int(val)
		case string:
			var parsed int
			_, err := fmt.Sscanf(val, "%d", &parsed)
			if err == nil {
				result[k] = parsed
			}
		default:
			var parsed int
			_, err := fmt.Sscanf(fmt.Sprintf("%v", val), "%d", &parsed)
			if err == nil {
				result[k] = parsed
			}
		}
	}
	return result, nil
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

// GetPathMappings reads pathMappings for a section ("radarr" or "sonarr") from config.yml and returns as [][]string
func GetPathMappings(section string) ([][]string, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}
	sec, _ := config[section].(map[string]interface{})
	var mappings [][]string
	if sec != nil {
		if pm, ok := sec["pathMappings"].([]interface{}); ok {
			for _, m := range pm {
				if mMap, ok := m.(map[string]interface{}); ok {
					from, _ := mMap["from"].(string)
					to, _ := mMap["to"].(string)
					if from != "" && to != "" {
						mappings = append(mappings, []string{from, to})
					}
				}
			}
		}
	}
	return mappings, nil
}

// Returns a Gin handler for settings (url, apiKey, pathMappings) for a given section ("radarr" or "sonarr")
// Returns url and apiKey for a given section (radarr/sonarr) from config.yml
func GetSectionUrlAndApiKey(section string) (string, string, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return "", "", err
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", "", err
	}
	sec, ok := config[section].(map[string]interface{})
	if !ok {
		return "", "", fmt.Errorf("section %s not found in config", section)
	}
	url, _ := sec["url"].(string)
	apiKey, _ := sec["apiKey"].(string)
	return url, apiKey, nil
}
func GetSettingsHandler(section string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := os.ReadFile(ConfigPath)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": ""})
			return
		}
		var config map[string]interface{}
		if err := yaml.Unmarshal(data, &config); err != nil {
			c.JSON(http.StatusOK, gin.H{"url": "", "apiKey": "", "pathMappings": []interface{}{}})
			return
		}
		sectionData, _ := config[section].(map[string]interface{})
		var mappings []map[string]string
		mappingSet := map[string]bool{}
		var pathMappings []map[string]interface{}
		if sectionData != nil {
			if pm, ok := sectionData["pathMappings"].([]interface{}); ok {
				for _, m := range pm {
					if mMap, ok := m.(map[string]interface{}); ok {
						from := ""
						to := ""
						if v, ok := mMap["from"].(string); ok {
							from = v
						}
						if v, ok := mMap["to"].(string); ok {
							to = v
						}
						mappings = append(mappings, map[string]string{"from": from, "to": to})
						mappingSet[from] = true
						pathMappings = append(pathMappings, map[string]interface{}{"from": from, "to": to})
					}
				}
			}
		}
		// Add any root folder from API response to settings if missing
		var folders []map[string]interface{}
		var url, apiKey string
		if sectionData != nil {
			url, _ = sectionData["url"].(string)
			apiKey, _ = sectionData["apiKey"].(string)
			if section == "radarr" {
				folders, _ = FetchRootFolders(url, apiKey)
			} else if section == "sonarr" {
				folders, _ = FetchRootFolders(url, apiKey)
			}
		}
		updated := false
		for _, f := range folders {
			if path, ok := f["path"].(string); ok {
				if !mappingSet[path] {
					pathMappings = append(pathMappings, map[string]interface{}{"from": path, "to": ""})
					mappings = append(mappings, map[string]string{"from": path, "to": ""})
					updated = true
				}
			}
		}
		if updated && sectionData != nil {
			sectionData["pathMappings"] = pathMappings
			config[section] = sectionData
			out, _ := yaml.Marshal(config)
			err := os.WriteFile(ConfigPath, out, 0644)
			if err != nil {
				fmt.Printf("[ERROR] Failed to save updated config: %v\n", err)
			} else {
				fmt.Printf("[INFO] Updated config with new root folders\n")
			}
		}
		c.JSON(http.StatusOK, gin.H{"url": url, "apiKey": apiKey, "pathMappings": mappings})
	}
}

// Returns a Gin handler to save settings (url, apiKey, pathMappings) for a given section ("radarr" or "sonarr")
func SaveSettingsHandler(section string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			URL          string `yaml:"url"`
			APIKey       string `yaml:"apiKey"`
			PathMappings []struct {
				From string `yaml:"from"`
				To   string `yaml:"to"`
			} `yaml:"pathMappings"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": ErrInvalidRequest})
			return
		}
		// Read existing config as map[string]interface{} to preserve all keys
		data, _ := os.ReadFile(ConfigPath)
		var config map[string]interface{}
		_ = yaml.Unmarshal(data, &config)
		// Update only the specified section
		sectionData := map[string]interface{}{
			"url":          req.URL,
			"apiKey":       req.APIKey,
			"pathMappings": req.PathMappings,
		}
		config[section] = sectionData
		out, _ := yaml.Marshal(config)
		err := os.WriteFile(ConfigPath, out, 0644)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "saved"})
	}
}

func getGeneralSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		c.JSON(http.StatusOK, gin.H{"tmdbKey": "", "autoDownloadExtras": true})
		return
	}
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	var tmdbKey string
	var autoDownloadExtras bool = true
	if general, ok := config["general"].(map[string]interface{}); ok {
		if v, ok := general["tmdbKey"].(string); ok {
			tmdbKey = v
		}
		if v, ok := general["autoDownloadExtras"].(bool); ok {
			autoDownloadExtras = v
		}
	}
	c.JSON(http.StatusOK, gin.H{"tmdbKey": tmdbKey, "autoDownloadExtras": autoDownloadExtras})
}

func saveGeneralSettingsHandler(c *gin.Context) {
	var req struct {
		TMDBApiKey         string `json:"tmdbKey" yaml:"tmdbKey"`
		AutoDownloadExtras *bool  `json:"autoDownloadExtras" yaml:"autoDownloadExtras"`
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
	var prevAutoDownload bool
	if v, ok := general["autoDownloadExtras"].(bool); ok {
		prevAutoDownload = v
	} else {
		prevAutoDownload = true
	}
	if req.AutoDownloadExtras != nil {
		general["autoDownloadExtras"] = *req.AutoDownloadExtras
		// Trigger start/stop of extras download task if changed
		if *req.AutoDownloadExtras && !prevAutoDownload {
			StartExtrasDownloadTask()
		} else if !*req.AutoDownloadExtras && prevAutoDownload {
			StopExtrasDownloadTask()
		}
	}
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
