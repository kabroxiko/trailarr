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
				if cfg.SearchMoviesExtras {
					fmt.Println("[TASK] Searching for missing movie extras...")
					DownloadMissingMoviesExtras()
				}
				if cfg.SearchSeriesExtras {
					fmt.Println("[TASK] Searching for missing series extras...")
					DownloadMissingSeriesExtras()
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
