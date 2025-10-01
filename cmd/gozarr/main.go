package main

import (
	"fmt"
	"net/http"
	"sort"
	"time"

	"gozarr/internal"

	"github.com/gin-gonic/gin"
)

const (
	TaskSyncWithRadarr = "Sync with Radarr"
	TaskSyncWithSonarr = "Sync with Sonarr"
)

var timings map[string]int

func getAllTasksStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build schedules array
		radarrInterval := fmt.Sprintf("%d minutes", internal.Timings["radarr"])
		sonarrInterval := fmt.Sprintf("%d minutes", internal.Timings["sonarr"])
		schedules := []gin.H{
			{
				"type":          TaskSyncWithRadarr,
				"name":          TaskSyncWithRadarr,
				"interval":      radarrInterval,
				"lastExecution": internal.RadarrLastExecution(),
				"lastDuration":  internal.RadarrLastDuration().String(),
				"nextExecution": internal.RadarrNextExecution(),
				"lastError":     internal.RadarrLastError(),
			},
			{
				"type":          TaskSyncWithSonarr,
				"name":          TaskSyncWithSonarr,
				"interval":      sonarrInterval,
				"lastExecution": internal.SonarrLastExecution(),
				"lastDuration":  internal.SonarrLastDuration().String(),
				"nextExecution": internal.SonarrNextExecution(),
				"lastError":     internal.SonarrLastError(),
			},
		}
		// Build queues array as []map[string]interface{} with type field
		queues := make([]map[string]interface{}, 0)
		for _, item := range internal.GlobalSyncQueue {
			var queueType string
			switch item.TaskName {
			case "radarr":
				queueType = TaskSyncWithRadarr
			case "sonarr":
				queueType = TaskSyncWithSonarr
			default:
				queueType = item.TaskName
			}
			queues = append(queues, map[string]interface{}{
				"type":     queueType,
				"Queued":   item.Queued,
				"Started":  item.Started,
				"Ended":    item.Ended,
				"Duration": item.Duration,
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

func main() {
	var err error
	timings, err = internal.EnsureSyncTimingsConfig()
	if err != nil {
		fmt.Printf("[WARN] Could not load sync timings: %v\n", err)
	}
	internal.Timings = timings
	fmt.Printf("[INFO] Sync timings: %v\n", timings)
	r := gin.Default()
	registerRoutes(r)
	startBackgroundTasks()
	r.Run(":8080")
}

func registerRoutes(r *gin.Engine) {
	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Register extras API endpoints
	internal.RegisterRoutes(r)

	// API endpoint for scheduled/queue status
	r.GET("/api/tasks/status", getAllTasksStatus())
	r.POST("/api/tasks/force", forceTaskHandler())

	// Serve React static files and SPA fallback
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
}

func forceTaskHandler() gin.HandlerFunc {
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
			go internal.ForceSyncRadarr()
			c.JSON(http.StatusOK, gin.H{"status": "Sync Radarr forced"})
		case TaskSyncWithSonarr:
			go internal.ForceSyncSonarr()
			c.JSON(http.StatusOK, gin.H{"status": "Sync Sonarr forced"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown task"})
		}
	}
}

func startBackgroundTasks() {
	go func() {
		interval := internal.Timings["radarr"]
		for {
			internal.ForceSyncRadarr()
			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()
	go func() {
		interval := internal.Timings["sonarr"]
		for {
			internal.ForceSyncSonarr()
			time.Sleep(time.Duration(interval) * time.Minute)
		}
	}()
}
