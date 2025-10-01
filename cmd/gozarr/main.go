package main

import (
       "gozarr/internal"
       "net/http"
       "time"

       "github.com/gin-gonic/gin"
)

const (
       TaskSyncWithRadarr         = "Sync with Radarr"
       TaskSyncWithRadarrInterval = "15 minutes"
       TaskSyncWithSonarr         = "Sync with Sonarr"
       TaskSyncWithSonarrInterval = "15 minutes"
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
				"type":          TaskSyncWithRadarr,
				"interval":      TaskSyncWithRadarrInterval,
				"lastExecution": syncRadarrStatus.LastExecution,
				"lastDuration":  syncRadarrStatus.LastDuration.String(),
				"nextExecution": syncRadarrStatus.NextExecution,
				"lastError":     syncRadarrStatus.LastError,
			},
			{
				"type":          TaskSyncWithSonarr,
				"name":          TaskSyncWithSonarr,
				"interval":      TaskSyncWithSonarrInterval,
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
				"type":     TaskSyncWithRadarr,
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
	r.GET("/api/tasks/sync-radarr/status", getRadarrStatusHandler())
	r.GET("/api/tasks/sync-sonarr/status", getSonarrStatusHandler())
	r.GET("/api/tasks/status", getAllTasksStatus())
	r.POST("/api/tasks/force", forceTaskHandler())

	// Serve React static files and SPA fallback
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})
}

func getRadarrStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          TaskSyncWithRadarr,
				"interval":      TaskSyncWithRadarrInterval,
				"lastExecution": syncRadarrStatus.LastExecution,
				"lastDuration":  syncRadarrStatus.LastDuration.String(),
				"nextExecution": syncRadarrStatus.NextExecution,
				"lastError":     syncRadarrStatus.LastError,
			},
			"queue": syncRadarrStatus.Queue,
		})
	}
}

func getSonarrStatusHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          TaskSyncWithSonarr,
				"interval":      TaskSyncWithSonarrInterval,
				"lastExecution": syncSonarrStatus.LastExecution,
				"lastDuration":  syncSonarrStatus.LastDuration.String(),
				"nextExecution": syncSonarrStatus.NextExecution,
				"lastError":     syncSonarrStatus.LastError,
			},
			"queue": syncSonarrStatus.Queue,
		})
	}
}

func forceTaskHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Name string `json:"name"`
		}
		if err := c.BindJSON(&req); err != nil {
			   c.JSON(http.StatusBadRequest, gin.H{"error": internal.ErrInvalidRequest})
			return
		}
		println("[FORCE] Requested force execution for:", req.Name)
		switch req.Name {
		case TaskSyncWithRadarr:
			go forceSyncRadarr()
			c.JSON(http.StatusOK, gin.H{"status": "Sync Radarr forced"})
		case TaskSyncWithSonarr:
			go forceSyncSonarr()
			c.JSON(http.StatusOK, gin.H{"status": "Sync Sonarr forced"})
		default:
			c.JSON(http.StatusBadRequest, gin.H{"error": "unknown task"})
		}
	}
}

func forceSyncRadarr() {
	println("[FORCE] Executing Sync Radarr...")
	item := SyncRadarrQueueItem{
		Queued: time.Now(),
		Status: "queued",
	}
	syncRadarrStatus.Queue = append(syncRadarrStatus.Queue, item)
	item.Started = time.Now()
	item.Status = "running"
	err := internal.SyncRadarrMoviesAndMediaCover()
	item.Ended = time.Now()
	item.Duration = item.Ended.Sub(item.Started)
	item.Status = "done"
	if err != nil {
		item.Error = err.Error()
		item.Status = "error"
		syncRadarrStatus.LastError = err.Error()
		println("[FORCE] Sync Radarr error:", err.Error())
	} else {
		println("[FORCE] Sync Radarr completed successfully.")
	}
	syncRadarrStatus.LastExecution = item.Ended
	syncRadarrStatus.LastDuration = item.Duration
	syncRadarrStatus.NextExecution = item.Ended.Add(15 * time.Minute)
	if len(syncRadarrStatus.Queue) > 10 {
		syncRadarrStatus.Queue = syncRadarrStatus.Queue[len(syncRadarrStatus.Queue)-10:]
	}
}

func forceSyncSonarr() {
	println("[FORCE] Executing Sync Sonarr...")
	item := SyncSonarrQueueItem{
		Queued: time.Now(),
		Status: "queued",
	}
	syncSonarrStatus.Queue = append(syncSonarrStatus.Queue, item)
	item.Started = time.Now()
	item.Status = "running"
	err := internal.SyncSonarrSeriesAndMediaCover()
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
		println("[FORCE] Sync Sonarr error:", err.Error())
	} else {
		println("[FORCE] Sync Sonarr completed successfully.")
	}
	syncSonarrStatus.LastExecution = item.Ended
	syncSonarrStatus.LastDuration = item.Duration
	syncSonarrStatus.NextExecution = item.Ended.Add(15 * time.Minute)
	if len(syncSonarrStatus.Queue) > 10 {
		syncSonarrStatus.Queue = syncSonarrStatus.Queue[len(syncSonarrStatus.Queue)-10:]
	}
}

func startBackgroundTasks() {
	go backgroundSyncRadarr()
	go backgroundSyncSonarr()
}

func backgroundSyncRadarr() {
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
		if len(syncRadarrStatus.Queue) > 10 {
			syncRadarrStatus.Queue = syncRadarrStatus.Queue[len(syncRadarrStatus.Queue)-10:]
		}
		<-ticker.C
	}
}

func backgroundSyncSonarr() {
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
		err := internal.SyncSonarrSeriesAndMediaCover()
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
}
