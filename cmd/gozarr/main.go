package main

import (
	"gozarr/internal"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	TaskSyncWithRadarr         = "Sync with Radarr"
	TaskSyncWithRadarrInterval = "15 minutes"
	TaskSyncWithSonarr         = "Sync with Sonarr"
	TaskSyncWithSonarrInterval = "15 minutes"
)

func getAllTasksStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build schedules array
		schedules := []gin.H{
			{
				"type":          TaskSyncWithRadarr,
				"name":          TaskSyncWithRadarr,
				"interval":      TaskSyncWithRadarrInterval,
				"lastExecution": internal.RadarrLastExecution(),
				"lastDuration":  internal.RadarrLastDuration().String(),
				"nextExecution": internal.RadarrNextExecution(),
				"lastError":     internal.RadarrLastError(),
			},
			{
				"type":          TaskSyncWithSonarr,
				"name":          TaskSyncWithSonarr,
				"interval":      TaskSyncWithSonarrInterval,
				"lastExecution": internal.SonarrLastExecution(),
				"lastDuration":  internal.SonarrLastDuration().String(),
				"nextExecution": internal.SonarrNextExecution(),
				"lastError":     internal.SonarrLastError(),
			},
		}
		// Build queues array as []map[string]interface{} with type field
		queues := make([]map[string]interface{}, 0)
		for _, item := range internal.RadarrQueue() {
			queues = append(queues, map[string]interface{}{
				"type":     TaskSyncWithRadarr,
				"Queued":   item.Queued,
				"Started":  item.Started,
				"Ended":    item.Ended,
				"Duration": item.Duration,
				"Status":   item.Status,
				"Error":    item.Error,
			})
		}
		for _, item := range internal.SonarrQueue() {
			queues = append(queues, map[string]interface{}{
				"type":     TaskSyncWithSonarr,
				"Queued":   item.Queued,
				"Started":  item.Started,
				"Ended":    item.Ended,
				"Duration": item.Duration,
				"Status":   item.Status,
				"Error":    item.Error,
			})
		}
		c.JSON(http.StatusOK, gin.H{
			"schedules": schedules,
			"queues":    queues,
		})
	}
}

func main() {
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
	go internal.BackgroundSyncRadarr()
	go internal.BackgroundSyncSonarr()
}
