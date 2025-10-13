package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"gopkg.in/yaml.v3"
)

const (
	TrailarrRoot             = "/var/lib/trailarr"
	ConfigPath               = TrailarrRoot + "/config/config.yml"
	MediaCoverPath           = TrailarrRoot + "/MediaCover"
	QueueFile                = TrailarrRoot + "/queue.json"
	CookiesFile              = TrailarrRoot + "/.config/google-chrome/cookies.txt"
	LogsDir                  = TrailarrRoot + "/logs"
	HistoryFile              = TrailarrRoot + "/history.json"
	MoviesJSONPath           = "trailarr:movies"
	SeriesJSONPath           = "trailarr:series"
	ExtrasCollectionKey      = "trailarr:extras"
	DownloadQueueRedisKey    = "trailarr:download_queue"
	ErrInvalidSonarrSettings = "Invalid Sonarr settings"
	RemoteMediaCoverPath     = "/MediaCover/"
	HeaderApiKey             = "X-Api-Key"
	ErrInvalidRequest        = "invalid request"
	HeaderContentType        = "Content-Type"
)

// Global in-memory config
var Config map[string]interface{}

// LoadConfig reads config.yml into the global Config variable
func LoadConfig() error {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return err
	}
	var cfg map[string]interface{}
	err = yaml.Unmarshal(data, &cfg)
	if err != nil {
		return err
	}
	Config = cfg
	return nil
}

// CanonicalizeExtraTypeConfig holds mapping from TMDB extra types to Plex extra types
type CanonicalizeExtraTypeConfig struct {
	Mapping map[string]string `yaml:"mapping" json:"mapping"`
}

// GetCanonicalizeExtraTypeConfig loads mapping config from config.yml
func GetCanonicalizeExtraTypeConfig() (CanonicalizeExtraTypeConfig, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return CanonicalizeExtraTypeConfig{Mapping: map[string]string{}}, err
	}
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	sec, ok := config["canonicalizeExtraType"].(map[string]interface{})
	cfg := CanonicalizeExtraTypeConfig{Mapping: map[string]string{}}
	if ok {
		if m, ok := sec["mapping"].(map[string]interface{}); ok {
			for k, v := range m {
				if s, ok := v.(string); ok {
					cfg.Mapping[k] = s
				}
			}
		}
	}
	return cfg, nil
}

// SaveCanonicalizeExtraTypeConfig saves mapping config to config.yml
func SaveCanonicalizeExtraTypeConfig(cfg CanonicalizeExtraTypeConfig) error {
	config, err := readConfigFile()
	if err != nil {
		config = map[string]interface{}{}
	}
	config["canonicalizeExtraType"] = map[string]interface{}{
		"mapping": cfg.Mapping,
	}
	return writeConfigFile(config)
}

// Handler to get canonicalizeExtraType config
func GetCanonicalizeExtraTypeConfigHandler(c *gin.Context) {
	cfg, _ := GetCanonicalizeExtraTypeConfig()
	respondJSON(c, http.StatusOK, cfg)
}

// Handler to save canonicalizeExtraType config
func SaveCanonicalizeExtraTypeConfigHandler(c *gin.Context) {
	var req CanonicalizeExtraTypeConfig
	if err := c.BindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	if err := SaveCanonicalizeExtraTypeConfig(req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(c, http.StatusOK, gin.H{"status": "saved"})
}

// Default values for config.yml
func DefaultGeneralConfig() map[string]interface{} {
	return map[string]interface{}{
		"tmdbKey":            "",
		"autoDownloadExtras": true,
		"logLevel":           "Debug",
	}
}

func EnsureConfigDefaults() error {
	var changed bool
	config, err := readConfigFileRaw()
	if err != nil {
		// Create file with all defaults
		config = map[string]interface{}{
			"general":    DefaultGeneralConfig(),
			"ytdlpFlags": DefaultYtdlpFlagsConfig(),
		}
		changed = true
	}
	// Fill in missing sections/keys even if file exists
	// General
	if config["general"] == nil {
		config["general"] = DefaultGeneralConfig()
		changed = true
	} else {
		general := config["general"].(map[string]interface{})
		for k, v := range DefaultGeneralConfig() {
			if _, ok := general[k]; !ok {
				general[k] = v
				changed = true
			}
		}
		config["general"] = general
	}
	// ytdlpFlags
	if config["ytdlpFlags"] == nil {
		config["ytdlpFlags"] = DefaultYtdlpFlagsConfig()
		changed = true
	} else {
		// Ensure cookiesFromBrowser is set to 'chrome' if not present
		ytdlpFlags, ok := config["ytdlpFlags"].(map[string]interface{})
		if ok {
			if _, ok := ytdlpFlags["cookiesFromBrowser"]; !ok {
				ytdlpFlags["cookiesFromBrowser"] = "chrome"
				config["ytdlpFlags"] = ytdlpFlags
				changed = true
			}
		}
	}
	// radarr
	if config["radarr"] == nil {
		config["radarr"] = map[string]interface{}{
			"url":          "http://localhost:7878",
			"apiKey":       "",
			"pathMappings": []map[string]string{},
		}
		changed = true
	} else {
		radarr := config["radarr"].(map[string]interface{})
		if _, ok := radarr["url"]; !ok {
			radarr["url"] = "http://localhost:7878"
			changed = true
		}
		if _, ok := radarr["apiKey"]; !ok {
			radarr["apiKey"] = ""
			changed = true
		}
		if _, ok := radarr["pathMappings"]; !ok {
			radarr["pathMappings"] = []map[string]string{}
			changed = true
		}
		config["radarr"] = radarr
	}
	// sonarr
	if config["sonarr"] == nil {
		config["sonarr"] = map[string]interface{}{
			"url":          "http://localhost:8989",
			"apiKey":       "",
			"pathMappings": []map[string]string{},
		}
		changed = true
	} else {
		sonarr := config["sonarr"].(map[string]interface{})
		if _, ok := sonarr["url"]; !ok {
			sonarr["url"] = "http://localhost:8989"
			changed = true
		}
		if _, ok := sonarr["apiKey"]; !ok {
			sonarr["apiKey"] = ""
			changed = true
		}
		if _, ok := sonarr["pathMappings"]; !ok {
			sonarr["pathMappings"] = []map[string]string{}
			changed = true
		}
		config["sonarr"] = sonarr
	}
	// extraTypes
	if config["extraTypes"] == nil {
		config["extraTypes"] = map[string]interface{}{
			"trailers":        true,
			"scenes":          true,
			"behindTheScenes": true,
			"interviews":      true,
			"featurettes":     true,
			"deletedScenes":   true,
			"other":           true,
		}
		changed = true
	} else {
		extraTypes := config["extraTypes"].(map[string]interface{})
		defaults := map[string]bool{
			"trailers":        true,
			"scenes":          true,
			"behindTheScenes": true,
			"interviews":      true,
			"featurettes":     true,
			"deletedScenes":   true,
			"other":           true,
		}
		for k, v := range defaults {
			if _, ok := extraTypes[k]; !ok {
				extraTypes[k] = v
				changed = true
			}
		}
		config["extraTypes"] = extraTypes
	}
	// Ensure canonicalizeExtraType mapping exists
	if config["canonicalizeExtraType"] == nil {
		// Default mapping: singular TMDB types to Plex types
		config["canonicalizeExtraType"] = map[string]interface{}{
			"mapping": map[string]string{
				"Trailer":          "Trailers",
				"Featurette":       "Featurettes",
				"Behind the Scene": "Behind The Scenes",
				"Deleted Scene":    "Deleted Scenes",
				"Interview":        "Interviews",
				"Scene":            "Scenes",
				"Short":            "Shorts",
				"Other":            "Other",
			},
		}
		changed = true
	}
	if changed {
		return writeConfigFile(config)
	}
	return nil
}

// Raw config file reader (no defaults)
func readConfigFileRaw() (map[string]interface{}, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	if len(data) == 0 {
		// Treat empty file as missing config
		return nil, fmt.Errorf("empty config file")
	}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// Helper to read config file and unmarshal into map[string]interface{}
func readConfigFile() (map[string]interface{}, error) {
	_ = EnsureConfigDefaults()
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return nil, err
	}
	var config map[string]interface{}
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return config, nil
}

// Helper to write config map to file
func writeConfigFile(config map[string]interface{}) error {
	out, err := yaml.Marshal(config)
	if err != nil {
		return err
	}
	return os.WriteFile(ConfigPath, out, 0644)
}

// EnsureYtdlpFlagsConfigExists checks config.yml and writes defaults if missing
func EnsureYtdlpFlagsConfigExists() error {
	config, err := readConfigFile()
	if err != nil {
		// If config file doesn't exist, create it with defaults
		config = map[string]interface{}{
			"ytdlpFlags": DefaultYtdlpFlagsConfig(),
		}
		return writeConfigFile(config)
	}
	if _, ok := config["ytdlpFlags"].(map[string]interface{}); !ok {
		// Add defaults if missing
		config["ytdlpFlags"] = DefaultYtdlpFlagsConfig()
		return writeConfigFile(config)
	}
	return nil
}

// --- YTDLP FLAGS CONFIG ---

// Loads yt-dlp flags config from config.yml
func GetYtdlpFlagsConfig() (YtdlpFlagsConfig, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return DefaultYtdlpFlagsConfig(), err
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return DefaultYtdlpFlagsConfig(), err
	}
	sec, ok := config["ytdlpFlags"].(map[string]interface{})
	cfg := DefaultYtdlpFlagsConfig()
	if !ok {
		return cfg, nil
	}
	// Map each field
	if v, ok := sec["quiet"].(bool); ok {
		cfg.Quiet = v
	}
	if v, ok := sec["noprogress"].(bool); ok {
		cfg.NoProgress = v
	}
	if v, ok := sec["cookiesFromBrowser"].(string); ok {
		cfg.CookiesFromBrowser = v
	}
	if v, ok := sec["writesubs"].(bool); ok {
		cfg.WriteSubs = v
	}
	if v, ok := sec["writeautosubs"].(bool); ok {
		cfg.WriteAutoSubs = v
	}
	if v, ok := sec["embedsubs"].(bool); ok {
		cfg.EmbedSubs = v
	}
	if v, ok := sec["remuxvideo"].(string); ok {
		cfg.RemuxVideo = v
	}
	if v, ok := sec["subformat"].(string); ok {
		cfg.SubFormat = v
	}
	if v, ok := sec["sublangs"].(string); ok {
		cfg.SubLangs = v
	}
	if v, ok := sec["requestedformats"].(string); ok {
		cfg.RequestedFormats = v
	}
	if v, ok := sec["timeout"].(float64); ok {
		cfg.Timeout = v
	} else if v, ok := sec["timeout"].(int); ok {
		cfg.Timeout = float64(v)
	}
	if v, ok := sec["sleepInterval"].(float64); ok {
		cfg.SleepInterval = v
	} else if v, ok := sec["sleepInterval"].(int); ok {
		cfg.SleepInterval = float64(v)
	}
	if v, ok := sec["maxDownloads"].(int); ok {
		cfg.MaxDownloads = v
	} else if v, ok := sec["maxDownloads"].(float64); ok {
		cfg.MaxDownloads = int(v)
	}
	if v, ok := sec["limitRate"].(string); ok {
		cfg.LimitRate = v
	}
	if v, ok := sec["sleepRequests"].(float64); ok {
		cfg.SleepRequests = v
	} else if v, ok := sec["sleepRequests"].(int); ok {
		cfg.SleepRequests = float64(v)
	}
	if v, ok := sec["maxSleepInterval"].(float64); ok {
		cfg.MaxSleepInterval = v
	} else if v, ok := sec["maxSleepInterval"].(int); ok {
		cfg.MaxSleepInterval = float64(v)
	}
	return cfg, nil
}

// Saves yt-dlp flags config to config.yml
func SaveYtdlpFlagsConfig(cfg YtdlpFlagsConfig) error {
	config, err := readConfigFile()
	if err != nil {
		config = map[string]interface{}{}
	}
	config["ytdlpFlags"] = map[string]interface{}{
		"quiet":              cfg.Quiet,
		"noprogress":         cfg.NoProgress,
		"writesubs":          cfg.WriteSubs,
		"writeautosubs":      cfg.WriteAutoSubs,
		"embedsubs":          cfg.EmbedSubs,
		"remuxvideo":         cfg.RemuxVideo,
		"subformat":          cfg.SubFormat,
		"sublangs":           cfg.SubLangs,
		"requestedformats":   cfg.RequestedFormats,
		"timeout":            cfg.Timeout,
		"sleepInterval":      cfg.SleepInterval,
		"maxDownloads":       cfg.MaxDownloads,
		"limitRate":          cfg.LimitRate,
		"sleepRequests":      cfg.SleepRequests,
		"maxSleepInterval":   cfg.MaxSleepInterval,
		"cookiesFromBrowser": cfg.CookiesFromBrowser,
	}
	return writeConfigFile(config)
}

// Handler to get yt-dlp flags config
func GetYtdlpFlagsConfigHandler(c *gin.Context) {
	cfg, err := GetYtdlpFlagsConfig()
	if err != nil {
		respondJSON(c, http.StatusOK, cfg)
		return
	}
	respondJSON(c, http.StatusOK, cfg)
}

// Handler to save yt-dlp flags config
func SaveYtdlpFlagsConfigHandler(c *gin.Context) {
	var req YtdlpFlagsConfig
	if err := c.BindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	if err := SaveYtdlpFlagsConfig(req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(c, http.StatusOK, gin.H{"status": "saved"})
}

var Timings map[string]int

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
	if Config == nil {
		return ExtraTypesConfig{Trailers: true, Scenes: true, BehindTheScenes: true, Interviews: true, Featurettes: true, DeletedScenes: true, Other: true}, nil
	}
	sec, ok := Config["extraTypes"].(map[string]interface{})
	cfg := ExtraTypesConfig{}
	if !ok {
		return ExtraTypesConfig{Trailers: true, Scenes: true, BehindTheScenes: true, Interviews: true, Featurettes: true, DeletedScenes: true, Other: true}, nil
	}
	if v, ok := sec["trailers"].(bool); ok {
		cfg.Trailers = v
	} else {
		cfg.Trailers = false
	}
	if v, ok := sec["scenes"].(bool); ok {
		cfg.Scenes = v
	} else {
		cfg.Scenes = false
	}
	if v, ok := sec["behindTheScenes"].(bool); ok {
		cfg.BehindTheScenes = v
	} else {
		cfg.BehindTheScenes = false
	}
	if v, ok := sec["interviews"].(bool); ok {
		cfg.Interviews = v
	} else {
		cfg.Interviews = false
	}
	if v, ok := sec["featurettes"].(bool); ok {
		cfg.Featurettes = v
	} else {
		cfg.Featurettes = false
	}
	if v, ok := sec["deletedScenes"].(bool); ok {
		cfg.DeletedScenes = v
	} else {
		cfg.DeletedScenes = false
	}
	if v, ok := sec["other"].(bool); ok {
		cfg.Other = v
	} else {
		cfg.Other = false
	}
	return cfg, nil
}

// SaveExtraTypesConfig saves extra types config to config.yml
func SaveExtraTypesConfig(cfg ExtraTypesConfig) error {
	config, err := readConfigFile()
	if err != nil {
		config = map[string]interface{}{}
	}
	config["extraTypes"] = map[string]interface{}{
		"trailers":        cfg.Trailers,
		"scenes":          cfg.Scenes,
		"behindTheScenes": cfg.BehindTheScenes,
		"interviews":      cfg.Interviews,
		"featurettes":     cfg.Featurettes,
		"deletedScenes":   cfg.DeletedScenes,
		"other":           cfg.Other,
	}
	err = writeConfigFile(config)
	if err == nil {
		// Update in-memory config
		if Config != nil {
			Config["extraTypes"] = config["extraTypes"]
		}
	}
	return err
}

// Handler to get extra types config
func GetExtraTypesConfigHandler(c *gin.Context) {
	cfg, err := GetExtraTypesConfig()
	if err != nil {
		respondJSON(c, http.StatusOK, cfg)
		return
	}
	respondJSON(c, http.StatusOK, cfg)
}

// Handler to save extra types config
func SaveExtraTypesConfigHandler(c *gin.Context) {
	var req ExtraTypesConfig
	if err := c.BindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	if err := SaveExtraTypesConfig(req); err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(c, http.StatusOK, gin.H{"status": "saved"})
}

// EnsureSyncTimingsConfig creates config.yml with sync timings if not present, or loads timings if present
func EnsureSyncTimingsConfig() (map[string]int, error) {
	defaultTimings := map[string]int{
		"radarr": 15,
		"sonarr": 15,
		"extras": 360,
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
	if !ok {
		// Add syncTimings without touching other config
		cfg["syncTimings"] = defaultTimings
		out, err := yaml.Marshal(cfg)
		if err == nil {
			_ = os.WriteFile(ConfigPath, out, 0644)
		}
		return defaultTimings, nil
	}
	// Ensure 'extras' interval is present
	if _, hasExtras := timings["extras"]; !hasExtras {
		timings["extras"] = 360
		cfg["syncTimings"] = timings
		out, err := yaml.Marshal(cfg)
		if err == nil {
			_ = os.WriteFile(ConfigPath, out, 0644)
		}
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
		TrailarrLog(WARN, "Settings", "settings not found: %v", err)
		return MediaSettings{}, fmt.Errorf("settings not found: %w", err)
	}
	var allSettings map[string]interface{}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		TrailarrLog(WARN, "Settings", "invalid settings: %v", err)
		return MediaSettings{}, fmt.Errorf("invalid settings: %w", err)
	}
	secRaw, ok := allSettings[section]
	if !ok {
		TrailarrLog(WARN, "Settings", "section %s not found", section)
		return MediaSettings{}, fmt.Errorf("section %s not found", section)
	}
	sec, ok := secRaw.(map[string]interface{})
	if !ok {
		TrailarrLog(WARN, "Settings", "section %s is not a map", section)
		return MediaSettings{}, fmt.Errorf("section %s is not a map", section)
	}
	var url, apiKey string
	if v, ok := sec["url"].(string); ok {
		url = v
	}
	if v, ok := sec["apiKey"].(string); ok {
		apiKey = v
	}
	return MediaSettings{URL: url, APIKey: apiKey}, nil
}

// GetPathMappings reads pathMappings for a section ("radarr" or "sonarr") from config.yml and returns as [][]string
func GetPathMappings(mediaType MediaType) ([][]string, error) {
	section := "radarr"
	if mediaType == MediaTypeTV {
		section = "sonarr"
	}
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
func GetProviderUrlAndApiKey(provider string) (string, string, error) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		return "", "", err
	}
	var config map[string]interface{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		return "", "", err
	}
	sec, ok := config[provider].(map[string]interface{})
	if !ok {
		TrailarrLog(WARN, "Settings", "section %s not found in config", provider)
		return "", "", fmt.Errorf("section %s not found in config", provider)
	}
	url, _ := sec["url"].(string)
	apiKey, _ := sec["apiKey"].(string)
	return url, apiKey, nil
}
func GetSettingsHandler(section string) gin.HandlerFunc {
	return func(c *gin.Context) {
		data, err := os.ReadFile(ConfigPath)
		if err != nil {
			respondJSON(c, http.StatusOK, gin.H{"url": "", "apiKey": ""})
			return
		}
		var config map[string]interface{}
		if err := yaml.Unmarshal(data, &config); err != nil {
			respondJSON(c, http.StatusOK, gin.H{"url": "", "apiKey": "", "pathMappings": []interface{}{}})
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
			switch section {
			case "radarr":
				folders, _ = FetchRootFolders(url, apiKey)
			case "sonarr":
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
				TrailarrLog(ERROR, "Settings", "Failed to save updated config: %v", err)
			} else {
				TrailarrLog(INFO, "Settings", "Updated config with new root folders")
			}
		}
		respondJSON(c, http.StatusOK, gin.H{"url": url, "apiKey": apiKey, "pathMappings": mappings})
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
			respondError(c, http.StatusBadRequest, ErrInvalidRequest)
			return
		}
		config, err := readConfigFile()
		if err != nil {
			config = map[string]interface{}{}
		}
		sectionData := map[string]interface{}{
			"url":          req.URL,
			"apiKey":       req.APIKey,
			"pathMappings": req.PathMappings,
		}
		config[section] = sectionData
		err = writeConfigFile(config)
		if err == nil {
			// Update in-memory config
			if Config != nil {
				Config[section] = sectionData
			}
		}
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(c, http.StatusOK, gin.H{"status": "saved"})
	}
}

func getGeneralSettingsHandler(c *gin.Context) {
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		respondJSON(c, http.StatusOK, gin.H{"tmdbKey": "", "autoDownloadExtras": true})
		return
	}
	var config map[string]interface{}
	_ = yaml.Unmarshal(data, &config)
	var tmdbKey string
	var autoDownloadExtras bool = true
	var logLevel string = "Debug"
	if general, ok := config["general"].(map[string]interface{}); ok {
		if v, ok := general["tmdbKey"].(string); ok {
			tmdbKey = v
		}
		if v, ok := general["autoDownloadExtras"].(bool); ok {
			autoDownloadExtras = v
		}
		if v, ok := general["logLevel"].(string); ok {
			logLevel = v
		}
	}
	respondJSON(c, http.StatusOK, gin.H{"tmdbKey": tmdbKey, "autoDownloadExtras": autoDownloadExtras, "logLevel": logLevel})
}

func saveGeneralSettingsHandler(c *gin.Context) {
	var req struct {
		TMDBApiKey         string `json:"tmdbKey" yaml:"tmdbKey"`
		AutoDownloadExtras *bool  `json:"autoDownloadExtras" yaml:"autoDownloadExtras"`
		LogLevel           string `json:"logLevel" yaml:"logLevel"`
	}
	if err := c.BindJSON(&req); err != nil {
		respondError(c, http.StatusBadRequest, ErrInvalidRequest)
		return
	}
	// Read existing settings as map[string]interface{} to preserve all keys
	config, err := readConfigFile()
	if err != nil {
		config = map[string]interface{}{}
	}
	if config["general"] == nil {
		config["general"] = map[string]interface{}{}
	}
	general := config["general"].(map[string]interface{})
	general["tmdbKey"] = req.TMDBApiKey
	// var prevAutoDownload bool
	// if v, ok := general["autoDownloadExtras"].(bool); ok {
	// 	prevAutoDownload = v
	// } else {
	// 	prevAutoDownload = true
	// }
	// if req.AutoDownloadExtras != nil {
	// 	general["autoDownloadExtras"] = *req.AutoDownloadExtras
	// 	// Trigger start/stop of extras download task if changed
	// 	if *req.AutoDownloadExtras && !prevAutoDownload {
	// 		StartExtrasDownloadTask()
	// 	} else if !*req.AutoDownloadExtras && prevAutoDownload {
	// 		StopExtrasDownloadTask()
	// 	}
	// }
	if req.LogLevel != "" {
		general["logLevel"] = req.LogLevel
	}
	config["general"] = general
	err = writeConfigFile(config)
	if err == nil {
		// Update in-memory config
		if Config != nil {
			Config["general"] = general
		}
	}
	if err != nil {
		respondError(c, http.StatusInternalServerError, err.Error())
		return
	}
	respondJSON(c, http.StatusOK, gin.H{"status": "saved"})
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
		TrailarrLog(WARN, "Settings", "API returned status %d", resp.StatusCode)
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

// Test connection to Radarr/Sonarr by calling /api/v3/system/status
func testMediaConnection(url, apiKey, _ string) error {
	endpoint := "/api/v3/system/status"
	req, err := http.NewRequest("GET", url+endpoint, nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-Api-Key", apiKey)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		TrailarrLog(WARN, "Settings", "API returned status %d", resp.StatusCode)
		return fmt.Errorf("API returned status %d", resp.StatusCode)
	}
	return nil
}

// Returns a slice of canonical extra types enabled in config
func GetEnabledCanonicalExtraTypes(cfg ExtraTypesConfig) []string {
	types := make([]string, 0)
	if cfg.Trailers {
		types = append(types, canonicalizeExtraType("trailers", ""))
	}
	if cfg.Scenes {
		types = append(types, canonicalizeExtraType("scenes", ""))
	}
	if cfg.BehindTheScenes {
		types = append(types, canonicalizeExtraType("behindTheScenes", ""))
	}
	if cfg.Interviews {
		types = append(types, canonicalizeExtraType("interviews", ""))
	}
	if cfg.Featurettes {
		types = append(types, canonicalizeExtraType("featurettes", ""))
	}
	if cfg.DeletedScenes {
		types = append(types, canonicalizeExtraType("deletedScenes", ""))
	}
	if cfg.Other {
		types = append(types, canonicalizeExtraType("other", ""))
	}
	if len(types) == 0 {
		types = []string{canonicalizeExtraType("trailers", "")}
	}
	return types
}
