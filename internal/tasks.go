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

const (
	TaskSyncWithRadarr = "Sync with Radarr"
	TaskSyncWithSonarr = "Sync with Sonarr"
)

func GetAllTasksStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		schedules := buildTaskSchedules()
		queues := buildTaskQueues()
		sortTaskQueuesByQueuedDesc(queues)
		c.JSON(http.StatusOK, gin.H{
			"schedules": schedules,
			"queues":    queues,
		})
	}
}

func buildTaskSchedules() []gin.H {
	radarrInterval := fmt.Sprintf("%d minutes", Timings["radarr"])
	sonarrInterval := fmt.Sprintf("%d minutes", Timings["sonarr"])
	return []gin.H{
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
}

func buildTaskQueues() []map[string]interface{} {
	queues := make([]map[string]interface{}, 0)
	for _, item := range GlobalSyncQueue {
		queueType := getQueueType(item.TaskName)
		startedOut := getTimeOrEmpty(item.Started)
		endedOut := getTimeOrEmpty(item.Ended)
		durationOut := getDurationOrEmpty(item.Duration)
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
	return queues
}

func getQueueType(taskName string) string {
	switch taskName {
	case "radarr":
		return TaskSyncWithRadarr
	case "sonarr":
		return TaskSyncWithSonarr
	default:
		return taskName
	}
}

func getTimeOrEmpty(t time.Time) interface{} {
	if !t.IsZero() {
		return t
	}
	return ""
}

func getDurationOrEmpty(d time.Duration) interface{} {
	if d > 0 {
		return d
	}
	return ""
}

func sortTaskQueuesByQueuedDesc(queues []map[string]interface{}) {
	sort.Slice(queues, func(i, j int) bool {
		qi, qj := queues[i]["Queued"], queues[j]["Queued"]
		ti, ok1 := qi.(time.Time)
		tj, ok2 := qj.(time.Time)
		if ok1 && ok2 {
			return ti.After(tj)
		}
		return false
	})
}

func TaskHandler() gin.HandlerFunc {
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
			go SyncRadarr()
			c.JSON(http.StatusOK, gin.H{"status": "Sync Radarr forced"})
		case TaskSyncWithSonarr:
			go SyncSonarr()
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
			SyncRadarr()
			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()
	go func() {
		interval := Timings["sonarr"]
		for {
			SyncSonarr()
			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()
	StartExtrasDownloadTask()
}

func StartExtrasDownloadTask() {
	StopExtrasDownloadTask()
	ctx, cancel := context.WithCancel(context.Background())
	extrasTaskCancel = cancel
	go func() {
		defer func() {}()
		for {
			if handleExtrasDownloadLoop(ctx) {
				return
			}
		}
	}()
}

func handleExtrasDownloadLoop(ctx context.Context) bool {
	if !GetAutoDownloadExtras() {
		fmt.Println("[TASK] Auto download of extras is disabled by general settings.")
		return waitOrDone(ctx, 360*time.Minute)
	}
	cfg := mustLoadSearchExtrasConfig()
	interval := getExtrasInterval()
	if cfg.AutoDownloadExtras {
		processExtras(cfg)
	} else {
		fmt.Println("[TASK] Auto download of extras is disabled by searchExtras config.")
	}
	return waitOrDone(ctx, time.Duration(interval)*time.Minute)
}

func waitOrDone(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		fmt.Println("[TASK] Extras download task stopped by cancel.")
		return true
	case <-time.After(d):
		return false
	}
}

func mustLoadSearchExtrasConfig() SearchExtrasConfig {
	cfg, err := GetSearchExtrasConfig()
	if err != nil {
		fmt.Printf("[WARN] Could not load search extras config: %v\n", err)
		cfg.SearchMoviesExtras = true
		cfg.SearchSeriesExtras = true
		cfg.AutoDownloadExtras = true
	}
	return cfg
}

func getExtrasInterval() int {
	interval := 360 // default 6 hours
	if v, ok := Timings["extras"]; ok {
		interval = v
	}
	return interval
}

func processExtras(cfg SearchExtrasConfig) {
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
		idInt, ok := parseMediaID(item["id"])
		if !ok {
			continue
		}
		extras, err := SearchExtras(mediaType, idInt)
		if err != nil {
			continue
		}
		mediaPath, err := FindMediaPathByID(cachePath, fmt.Sprintf("%v", item["id"]))
		if err != nil || mediaPath == "" {
			continue
		}
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		filterAndDownloadTypeFilteredExtras(cfg, mediaType, item, extras)
	}
}

func filterAndDownloadTypeFilteredExtras(cfg ExtraTypesConfig, mediaType string, item map[string]interface{}, extras []map[string]string) {
	for _, extra := range extras {
		typ := canonicalizeExtraType(extra["type"], extra["type"])
		if !isExtraTypeEnabled(cfg, typ) {
			continue
		}
		if extra["downloaded"] == "false" && extra["url"] != "" {
			err := handleTypeFilteredExtraDownload(mediaType, item, extra)
			if err != nil {
				fmt.Printf("[DownloadMissingExtrasWithTypeFilter] Failed to download: %v\n", err)
			}
		}
	}
}

func handleTypeFilteredExtraDownload(mediaType string, item map[string]interface{}, extra map[string]string) error {
	title, _ := item["title"].(string)
	_, err := DownloadYouTubeExtra(mediaType, title, extra["type"], extra["title"], extra["url"])
	return err
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
