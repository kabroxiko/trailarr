package main

import (
	"gozarr/internal"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	// Track last sync status and times
	syncRadarrStatus = struct {
		LastExecution time.Time
		LastDuration  time.Duration
		NextExecution time.Time
		LastError     string
		Queue         []SyncRadarrQueueItem
	}{
		Queue: make([]SyncRadarrQueueItem, 0),
	}
	syncSonarrStatus = struct {
		LastExecution time.Time
		LastDuration  time.Duration
		NextExecution time.Time
		LastError     string
		Queue         []SyncSonarrQueueItem
	}{
		Queue: make([]SyncSonarrQueueItem, 0),
	}
)

type SyncRadarrQueueItem struct {
	Queued   time.Time
	Started  time.Time
	Ended    time.Time
	Duration time.Duration
	Status   string
	Error    string
}

type SyncSonarrQueueItem struct {
	Queued   time.Time
	Started  time.Time
	Ended    time.Time
	Duration time.Duration
	Status   string
	Error    string
}

func getAllTasksStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Build schedules array
		schedules := []gin.H{
			{
				"type":          "Sync Radarr",
				"name":          "Sync Radarr",
				"interval":      "15 minutes",
				"lastExecution": syncRadarrStatus.LastExecution,
				"lastDuration":  syncRadarrStatus.LastDuration.String(),
				"nextExecution": syncRadarrStatus.NextExecution,
				"lastError":     syncRadarrStatus.LastError,
			},
			{
				"type":          "Sync Sonarr",
				"name":          "Sync Sonarr",
				"interval":      "15 minutes",
				"lastExecution": syncSonarrStatus.LastExecution,
				"lastDuration":  syncSonarrStatus.LastDuration.String(),
				"nextExecution": syncSonarrStatus.NextExecution,
				"lastError":     syncSonarrStatus.LastError,
			},
		}
		// Build queues array as []map[string]interface{} with type field
		queues := make([]map[string]interface{}, 0)
		for _, item := range syncRadarrStatus.Queue {
			queues = append(queues, map[string]interface{}{
				"type":     "Sync Radarr",
				"Queued":   item.Queued,
				"Started":  item.Started,
				"Ended":    item.Ended,
				"Duration": item.Duration,
				"Status":   item.Status,
				"Error":    item.Error,
			})
		}
		for _, item := range syncSonarrStatus.Queue {
			queues = append(queues, map[string]interface{}{
				"type":     "Sync Sonarr",
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

	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Register extras API endpoints
	internal.RegisterRoutes(r)

	// API endpoint for scheduled/queue status
	r.GET("/api/tasks/sync-radarr/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          "Sync Radarr",
				"interval":      "15 minutes",
				"lastExecution": syncRadarrStatus.LastExecution,
				"lastDuration":  syncRadarrStatus.LastDuration.String(),
				"nextExecution": syncRadarrStatus.NextExecution,
				"lastError":     syncRadarrStatus.LastError,
			},
			"queue": syncRadarrStatus.Queue,
		})
	})

	// API endpoint for scheduled/queue status (Sonarr)
	r.GET("/api/tasks/sync-sonarr/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          "Sync Sonarr",
				"interval":      "15 minutes",
				"lastExecution": syncSonarrStatus.LastExecution,
				"lastDuration":  syncSonarrStatus.LastDuration.String(),
				"nextExecution": syncSonarrStatus.NextExecution,
				"lastError":     syncSonarrStatus.LastError,
			},
			"queue": syncSonarrStatus.Queue,
		})
	})

	// API endpoint for combined scheduled/queue status
	r.GET("/api/tasks/status", getAllTasksStatus())

	// Background sync task: sync Radarr movies and MediaCover every 15 minutes
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			item := SyncRadarrQueueItem{
				Queued: time.Now(),
				Status: "queued",
			}
			syncRadarrStatus.Queue = append(syncRadarrStatus.Queue, item)
			item.Started = time.Now()
			item.Status = "running"
			// Run sync
			err := internal.SyncRadarrMoviesAndMediaCover()
			item.Ended = time.Now()
			item.Duration = item.Ended.Sub(item.Started)
			item.Status = "done"
			if err != nil {
				item.Error = err.Error()
				item.Status = "error"
				syncRadarrStatus.LastError = err.Error()
			}
			syncRadarrStatus.LastExecution = item.Ended
			syncRadarrStatus.LastDuration = item.Duration
			syncRadarrStatus.NextExecution = time.Now().Add(15 * time.Minute)
			// Keep only last 10 queue items
			if len(syncRadarrStatus.Queue) > 10 {
				syncRadarrStatus.Queue = syncRadarrStatus.Queue[len(syncRadarrStatus.Queue)-10:]
			}
			<-ticker.C
		}
	}()

	// Background sync task: sync Sonarr series every 15 minutes
	go func() {
		ticker := time.NewTicker(15 * time.Minute)
		defer ticker.Stop()
		for {
			item := SyncSonarrQueueItem{
				Queued: time.Now(),
				Status: "queued",
			}
			syncSonarrStatus.Queue = append(syncSonarrStatus.Queue, item)
			item.Started = time.Now()
			item.Status = "running"
			var err error
			err = internal.SyncSonarrSeriesAndMediaCover()
			item.Ended = time.Now()
			item.Duration = item.Ended.Sub(item.Started)
			if err == nil {
				item.Status = "done"
			} else {
				item.Status = "error"
			}
			if err != nil {
				item.Error = err.Error()
				syncSonarrStatus.LastError = err.Error()
			}
			syncSonarrStatus.LastExecution = item.Ended
			syncSonarrStatus.LastDuration = item.Duration
			syncSonarrStatus.NextExecution = time.Now().Add(15 * time.Minute)
			if len(syncSonarrStatus.Queue) > 10 {
				syncSonarrStatus.Queue = syncSonarrStatus.Queue[len(syncSonarrStatus.Queue)-10:]
			}
			<-ticker.C
		}
	}()

	// Serve React static files and SPA fallback
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	r.Run(":8080")
}
