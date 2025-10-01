package internal

import (
	"context"
	"fmt"
	"net/http"
	"sort"
	"time"

	"github.com/gin-gonic/gin"
)

var extrasTaskCancel context.CancelFunc
var extrasTaskRunning bool

const (
	TaskSyncWithRadarr = "Sync with Radarr"
	TaskSyncWithSonarr = "Sync with Sonarr"
)

func GetAllTasksStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build schedules array
		radarrInterval := fmt.Sprintf("%d minutes", Timings["radarr"])
		sonarrInterval := fmt.Sprintf("%d minutes", Timings["sonarr"])
		schedules := []gin.H{
			{
				"type":          TaskSyncWithRadarr,
				"name":          TaskSyncWithRadarr,
				"interval":      radarrInterval,
				"lastExecution": RadarrLastExecution(),
				"lastDuration":  RadarrLastDuration().String(),
				"nextExecution": RadarrNextExecution(),
				"lastError":     RadarrLastError(),
			},
			{
				"type":          TaskSyncWithSonarr,
				"name":          TaskSyncWithSonarr,
				"interval":      sonarrInterval,
				"lastExecution": SonarrLastExecution(),
				"lastDuration":  SonarrLastDuration().String(),
				"nextExecution": SonarrNextExecution(),
				"lastError":     SonarrLastError(),
			},
		}
		// Build queues array as []map[string]interface{} with type field
		queues := make([]map[string]interface{}, 0)
		for _, item := range GlobalSyncQueue {
			var queueType string
			switch item.TaskName {
			case "radarr":
				queueType = TaskSyncWithRadarr
			case "sonarr":
				queueType = TaskSyncWithSonarr
			default:
				queueType = item.TaskName
			}
			// Patch: convert zero Ended and zero Duration to empty string for frontend compatibility
			var startedOut interface{} = ""
			if !item.Started.IsZero() {
				startedOut = item.Started
			}
			var endedOut interface{} = ""
			if !item.Ended.IsZero() {
				endedOut = item.Ended
			}
			var durationOut interface{} = ""
			if item.Duration > 0 {
				durationOut = item.Duration
			}
			queues = append(queues, map[string]interface{}{
				"type":     queueType,
				"Queued":   item.Queued,
				"Started":  startedOut,
				"Ended":    endedOut,
				"Duration": durationOut,
				"Status":   item.Status,
				"Error":    item.Error,
			})
		}
		// Sort queues by Queued date descending
		sort.Slice(queues, func(i, j int) bool {
			qi, qj := queues[i]["Queued"], queues[j]["Queued"]
			ti, ok1 := qi.(time.Time)
			tj, ok2 := qj.(time.Time)
			if ok1 && ok2 {
				return ti.After(tj)
			}
			return false
		})
		c.JSON(http.StatusOK, gin.H{
			"schedules": schedules,
			"queues":    queues,
		})
	}
}

func ForceTaskHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		if err := c.BindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
			return
		}
		println("[FORCE] Requested force execution for:", req.Name)
		switch req.Name {
		case TaskSyncWithRadarr:
			go ForceSyncRadarr()
			c.JSON(http.StatusOK, gin.H{"status": "Sync Radarr forced"})
		case TaskSyncWithSonarr:
			go ForceSyncSonarr()
			c.JSON(http.StatusOK, gin.H{"status": "Sync Sonarr forced"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown task"})
		}
	}
}

func StartBackgroundTasks() {
	go func() {
		interval := Timings["radarr"]
		for {
			ForceSyncRadarr()
			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()
	go func() {
		interval := Timings["sonarr"]
		for {
			ForceSyncSonarr()
			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()
	StartExtrasDownloadTask()
}

func StartExtrasDownloadTask() {
	StopExtrasDownloadTask()
	extrasTaskRunning = false
	ctx, cancel := context.WithCancel(context.Background())
	extrasTaskCancel = cancel
	extrasTaskRunning = true
	go func() {
		defer func() { extrasTaskRunning = false }()
		for {
			if !GetAutoDownloadExtras() {
				fmt.Println("[TASK] Auto download of extras is disabled by general settings.")
				select {
				case <-ctx.Done():
					fmt.Println("[TASK] Extras download task stopped by cancel.")
					return
				case <-time.After(360 * time.Minute):
				}
				continue
			}
			cfg, err := GetSearchExtrasConfig()
			if err != nil {
				fmt.Printf("[WARN] Could not load search extras config: %v\n", err)
				cfg.SearchMoviesExtras = true
				cfg.SearchSeriesExtras = true
				cfg.AutoDownloadExtras = true
			}
			interval := 360 // default 6 hours
			if v, ok := Timings["extras"]; ok {
				interval = v
			}
			if cfg.AutoDownloadExtras {
				extraTypesCfg, err := GetExtraTypesConfig()
				if err != nil {
					fmt.Printf("[WARN] Could not load extra types config: %v\n", err)
				}
				if cfg.SearchMoviesExtras {
					fmt.Println("[TASK] Searching for missing movie extras...")
					DownloadMissingMoviesExtrasWithTypeFilter(extraTypesCfg)
				}
				if cfg.SearchSeriesExtras {
					fmt.Println("[TASK] Searching for missing series extras...")
					DownloadMissingSeriesExtrasWithTypeFilter(extraTypesCfg)
				}
			} else {
				fmt.Println("[TASK] Auto download of extras is disabled by searchExtras config.")
			}
			select {
			case <-ctx.Done():
				fmt.Println("[TASK] Extras download task stopped by cancel.")
				return
			case <-time.After(time.Duration(interval) * time.Minute):
			}
		}
	}()
}

func StopExtrasDownloadTask() {
	if extrasTaskCancel != nil {
		extrasTaskCancel()
		extrasTaskCancel = nil
	}
}

// Download missing movie extras, filtering by enabled types
func DownloadMissingMoviesExtrasWithTypeFilter(cfg ExtraTypesConfig) {
	// Example: get all movies, for each, get extras, filter by type, download only enabled types
	downloadMissingExtrasWithTypeFilter(cfg, "movie", TrailarrRoot+"/movies_wanted.json")
}

// Download missing series extras, filtering by enabled types
func DownloadMissingSeriesExtrasWithTypeFilter(cfg ExtraTypesConfig) {
	downloadMissingExtrasWithTypeFilter(cfg, "tv", TrailarrRoot+"/series_wanted.json")
}

// Shared logic for type-filtered extras download
func downloadMissingExtrasWithTypeFilter(cfg ExtraTypesConfig, mediaType, cachePath string) {
	items, err := loadCache(cachePath)
	if err != nil {
		fmt.Printf("[DEBUG] Failed to load cache: %v\n", err)
		return
	}
	for _, item := range items {
		id, ok := item["id"]
		if !ok {
			continue
		}
		var idInt int
		switch v := id.(type) {
		case int:
			idInt = v
		case float64:
			idInt = int(v)
		case string:
			fmt.Sscanf(v, "%d", &idInt)
		}
		extras, err := SearchExtras(mediaType, idInt)
		if err != nil {
			continue
		}
		mediaPath, err := FindMediaPathByID(cachePath, fmt.Sprintf("%v", id))
		if err != nil || mediaPath == "" {
			continue
		}
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		for _, extra := range extras {
			typ := canonicalizeExtraType(extra["type"], extra["type"])
			if !isExtraTypeEnabled(cfg, typ) {
				continue
			}
			if extra["downloaded"] == "false" && extra["url"] != "" {
				_, err := DownloadYouTubeExtra(mediaType, item["title"].(string), extra["type"], extra["title"], extra["url"])
				if err != nil {
					fmt.Printf("[DownloadMissingExtrasWithTypeFilter] Failed to download: %v\n", err)
				}
			}
		}
	}
}

// Helper: check if extra type is enabled in config
func isExtraTypeEnabled(cfg ExtraTypesConfig, typ string) bool {
	switch typ {
	case "Trailers":
		return cfg.Trailers
	case "Scenes":
		return cfg.Scenes
	case "Behind The Scenes":
		return cfg.BehindTheScenes
	case "Interviews":
		return cfg.Interviews
	case "Featurettes":
		return cfg.Featurettes
	case "Deleted Scenes":
		return cfg.DeletedScenes
	case "Others", "Other":
		return cfg.Other
	default:
		return false
	}
}
