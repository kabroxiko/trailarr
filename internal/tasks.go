package internal

import (
	"context"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"time"

	scheduler "github.com/algorythma/go-scheduler"
	"github.com/algorythma/go-scheduler/storage"
	"github.com/algorythma/go-scheduler/task"
	"github.com/gin-gonic/gin"
)

var GlobalTaskTimes TaskTimes

var extrasTaskCancel context.CancelFunc
var extrasTaskStarted bool // DEBUG: track all changes and reads
var radarrTaskStarted bool
var sonarrTaskStarted bool

const (
	TaskSyncWithRadarr = "Sync with Radarr"
	TaskSyncWithSonarr = "Sync with Sonarr"
)

const taskTimesFile = "task_times.json"

type Task struct {
	TaskId        string    `json:"taskId"`
	Name          string    `json:"-"`
	Interval      int       `json:"interval"` // minutes
	LastExecution time.Time `json:"lastExecution"`
	LastDuration  float64   `json:"lastDuration"` // seconds
	// NextExecution is not persisted; calculated dynamically
}

type TaskTimes struct {
	Radarr Task `json:"radarr"`
	Sonarr Task `json:"sonarr"`
	Extras Task `json:"extras"`
}

func LoadTaskTimes() (TaskTimes, error) {
	var arr []Task
	path := filepath.Join(TrailarrRoot, taskTimesFile)
	err := ReadJSONFile(path, &arr)
	var times TaskTimes
	if err != nil {
		// If file does not exist, initialize with empty times and create file
		if os.IsNotExist(err) {
			times = TaskTimes{
				Radarr: Task{TaskId: "radarr"},
				Sonarr: Task{TaskId: "sonarr"},
				Extras: Task{TaskId: "extras"},
			}
			_ = saveTaskTimes(times)
			GlobalTaskTimes = times
			return times, nil
		}
		return times, err
	}
	// Map array entries to struct fields by TaskId
	for _, t := range arr {
		switch t.TaskId {
		case "radarr":
			times.Radarr = t
		case "sonarr":
			times.Sonarr = t
		case "extras":
			times.Extras = t
		}
	}
	GlobalTaskTimes = times
	return times, nil
}

func saveTaskTimes(times TaskTimes) error {
	GlobalTaskTimes = times
	arr := []Task{times.Radarr, times.Sonarr, times.Extras}
	return WriteJSONFile(filepath.Join(TrailarrRoot, taskTimesFile), arr)
}

func GetAllTasksStatus() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Use running flags for Radarr/Sonarr
		var radarrStatus, sonarrStatus, extrasStatus string
		if radarrTaskStarted {
			radarrStatus = "running"
		} else {
			radarrStatus = "idle"
		}
		if sonarrTaskStarted {
			sonarrStatus = "running"
		} else {
			sonarrStatus = "idle"
		}
		if extrasTaskStarted {
			extrasStatus = "running"
		} else {
			extrasStatus = "idle"
		}
		times := GlobalTaskTimes
		calcNext := func(lastExecution time.Time, interval int) time.Time {
			if lastExecution.IsZero() {
				return time.Now().Add(time.Duration(interval) * time.Minute)
			}
			return lastExecution.Add(time.Duration(interval) * time.Minute)
		}
		// Compute status for each scheduled task
		// Set TaskId and Name for each schedule (runtime only)
		radarrTaskId := "radarr"
		radarrName := TaskSyncWithRadarr
		sonarrTaskId := "sonarr"
		sonarrName := TaskSyncWithSonarr
		extrasTaskId := "extras"
		extrasName := "Search for Missing Extras"

		schedules := []map[string]interface{}{
			{
				"taskId":        radarrTaskId,
				"name":          radarrName,
				"interval":      times.Radarr.Interval,
				"lastExecution": getTimeOrEmpty(times.Radarr.LastExecution),
				"lastDuration":  times.Radarr.LastDuration,
				"nextExecution": calcNext(times.Radarr.LastExecution, times.Radarr.Interval),
				"status":        radarrStatus,
			},
			{
				"taskId":        sonarrTaskId,
				"name":          sonarrName,
				"interval":      times.Sonarr.Interval,
				"lastExecution": getTimeOrEmpty(times.Sonarr.LastExecution),
				"lastDuration":  times.Sonarr.LastDuration,
				"nextExecution": calcNext(times.Sonarr.LastExecution, times.Sonarr.Interval),
				"status":        sonarrStatus,
			},
			{
				"taskId":        extrasTaskId,
				"name":          extrasName,
				"interval":      times.Extras.Interval,
				"lastExecution": getTimeOrEmpty(times.Extras.LastExecution),
				"lastDuration":  times.Extras.LastDuration,
				"nextExecution": calcNext(times.Extras.LastExecution, times.Extras.Interval),
				"status":        extrasStatus,
			},
		}
		queues := buildTaskQueues()
		sortTaskQueuesByQueuedDesc(queues)
		respondJSON(c, http.StatusOK, gin.H{
			"schedules": schedules,
			"queues":    queues,
		})
	}
}

func buildTaskQueues() []map[string]interface{} {
	queues := make([]map[string]interface{}, 0)
	for _, item := range GlobalSyncQueue {
		queues = append(queues, NewQueueStatusMap(item))
	}
	return queues
}

// NewQueueStatusMap constructs a status map for a SyncQueueItem
func NewQueueStatusMap(item SyncQueueItem) map[string]interface{} {
	return map[string]interface{}{
		"type":     getQueueType(item.TaskName),
		"Queued":   item.Queued,
		"Started":  getTimeOrEmpty(item.Started),
		"Ended":    getTimeOrEmpty(item.Ended),
		"Duration": getDurationOrEmpty(item.Duration),
		"Status":   item.Status,
		"Error":    item.Error,
	}
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
			TaskId string `json:"taskId"`
		}
		if err := c.BindJSON(&req); err != nil {
			respondError(c, http.StatusBadRequest, "invalid request")
			return
		}
		println("[FORCE] Requested force execution for:", req.TaskId)
		times, _ := LoadTaskTimes()
		switch req.TaskId {
		case TaskSyncWithRadarr, "radarr":
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Starting Radarr job at %s", time.Now().Format(time.RFC3339Nano))
			radarrTaskStarted = true
			// Only broadcast Radarr as running
			calcNext := func(lastExecution time.Time, interval int) time.Time {
				if lastExecution.IsZero() {
					return time.Now().Add(time.Duration(interval) * time.Minute)
				}
				return lastExecution.Add(time.Duration(interval) * time.Minute)
			}
			// extrasName variable removed; name is now hardcoded in broadcast
			broadcastTaskStatus(map[string]interface{}{
				"schedules": []map[string]interface{}{
					{
						"taskId":        "radarr",
						"name":          "Sync with Radarr",
						"interval":      times.Radarr.Interval,
						"lastExecution": getTimeOrEmpty(times.Radarr.LastExecution),
						"lastDuration":  times.Radarr.LastDuration,
						"nextExecution": calcNext(times.Radarr.LastExecution, times.Radarr.Interval),
						"status":        "running",
					},
					{
						"taskId":        "sonarr",
						"name":          "Sync with Sonarr",
						"interval":      times.Sonarr.Interval,
						"lastExecution": getTimeOrEmpty(times.Sonarr.LastExecution),
						"lastDuration":  times.Sonarr.LastDuration,
						"nextExecution": calcNext(times.Sonarr.LastExecution, times.Sonarr.Interval),
						"status":        "idle",
					},
					{
						"taskId":        "extras",
						"name":          "Search for Missing Extras",
						"interval":      times.Extras.Interval,
						"lastExecution": getTimeOrEmpty(times.Extras.LastExecution),
						"lastDuration":  times.Extras.LastDuration,
						"nextExecution": calcNext(times.Extras.LastExecution, times.Extras.Interval),
						"status":        "idle",
					},
				},
			})
			start := time.Now()
			SyncRadarr()
			duration := time.Since(start)
			radarrTaskStarted = false
			broadcastTaskStatus(getCurrentTaskStatus())
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Radarr job finished at %s, duration=%s", time.Now().Format(time.RFC3339Nano), duration.String())
			times.Radarr.LastDuration = duration.Seconds()
			times.Radarr.LastExecution = start
			saveTaskTimes(times)
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Updated times.Radarr: %+v", times.Radarr)
			respondJSON(c, http.StatusOK, gin.H{"status": "Sync Radarr forced"})
		case TaskSyncWithSonarr, "sonarr":
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Starting Sonarr job at %s", time.Now().Format(time.RFC3339Nano))
			sonarrTaskStarted = true
			calcNext := func(lastExecution time.Time, interval int) time.Time {
				if lastExecution.IsZero() {
					return time.Now().Add(time.Duration(interval) * time.Minute)
				}
				return lastExecution.Add(time.Duration(interval) * time.Minute)
			}
			broadcastTaskStatus(map[string]interface{}{
				"schedules": []map[string]interface{}{
					{
						"taskId":        "radarr",
						"name":          TaskSyncWithRadarr,
						"interval":      times.Radarr.Interval,
						"lastExecution": getTimeOrEmpty(times.Radarr.LastExecution),
						"lastDuration":  times.Radarr.LastDuration,
						"nextExecution": calcNext(times.Radarr.LastExecution, times.Radarr.Interval),
						"status":        "idle",
					},
					{
						"taskId":        "sonarr",
						"name":          TaskSyncWithSonarr,
						"interval":      times.Sonarr.Interval,
						"lastExecution": getTimeOrEmpty(times.Sonarr.LastExecution),
						"lastDuration":  times.Sonarr.LastDuration,
						"nextExecution": calcNext(times.Sonarr.LastExecution, times.Sonarr.Interval),
						"status":        "running",
					},
					{
						"taskId":        "extras",
						"name":          "Search for Missing Extras",
						"interval":      times.Extras.Interval,
						"lastExecution": getTimeOrEmpty(times.Extras.LastExecution),
						"lastDuration":  times.Extras.LastDuration,
						"nextExecution": calcNext(times.Extras.LastExecution, times.Extras.Interval),
						"status":        "idle",
					},
				},
			})
			start := time.Now()
			SyncSonarr()
			duration := time.Since(start)
			sonarrTaskStarted = false
			broadcastTaskStatus(getCurrentTaskStatus())
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Sonarr job finished at %s, duration=%s", time.Now().Format(time.RFC3339Nano), duration.String())
			times.Sonarr.LastDuration = duration.Seconds()
			times.Sonarr.LastExecution = start
			saveTaskTimes(times)
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Updated times.Sonarr: %+v", times.Sonarr)
			respondJSON(c, http.StatusOK, gin.H{"status": "Sync Sonarr forced"})
		case "extras":
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Starting Extras job at %s", time.Now().Format(time.RFC3339Nano))
			// Do not set extrasTaskStarted here; StartExtrasDownloadTask will set it if the task actually starts
			calcNext := func(lastExecution time.Time, interval int) time.Time {
				if lastExecution.IsZero() {
					return time.Now().Add(time.Duration(interval) * time.Minute)
				}
				return lastExecution.Add(time.Duration(interval) * time.Minute)
			}
			broadcastTaskStatus(map[string]interface{}{
				"schedules": []map[string]interface{}{
					{
						"taskId":        "radarr",
						"name":          TaskSyncWithRadarr,
						"interval":      times.Radarr.Interval,
						"lastExecution": getTimeOrEmpty(times.Radarr.LastExecution),
						"lastDuration":  times.Radarr.LastDuration,
						"nextExecution": calcNext(times.Radarr.LastExecution, times.Radarr.Interval),
						"status":        "idle",
					},
					{
						"taskId":        "sonarr",
						"name":          TaskSyncWithSonarr,
						"interval":      times.Sonarr.Interval,
						"lastExecution": getTimeOrEmpty(times.Sonarr.LastExecution),
						"lastDuration":  times.Sonarr.LastDuration,
						"nextExecution": calcNext(times.Sonarr.LastExecution, times.Sonarr.Interval),
						"status":        "idle",
					},
					{
						"taskId":        "extras",
						"name":          "Search for Missing Extras",
						"interval":      times.Extras.Interval,
						"lastExecution": getTimeOrEmpty(times.Extras.LastExecution),
						"lastDuration":  times.Extras.LastDuration,
						"nextExecution": calcNext(times.Extras.LastExecution, times.Extras.Interval),
						"status":        "running",
					},
				},
			})
			start := time.Now()
			StartExtrasDownloadTask()
			duration := time.Since(start)
			// Do not reset extrasTaskStarted here; let the goroutine handle it
			broadcastTaskStatus(getCurrentTaskStatus())
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Extras job finished at %s, duration=%s", time.Now().Format(time.RFC3339Nano), duration.String())
			times.Extras.LastDuration = duration.Seconds()
			times.Extras.LastExecution = start
			saveTaskTimes(times)
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Updated times.Extras: %+v", times.Extras)
			respondJSON(c, http.StatusOK, gin.H{"status": "Sync Extras forced"})
		default:
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Unknown job requested: %s", req.TaskId)
			respondError(c, http.StatusBadRequest, "unknown task")
		}
	}
}

func StartBackgroundTasks() {
	TrailarrLog(INFO, "Tasks", "StartBackgroundTasks called. PID=%d, time=%s", os.Getpid(), time.Now().Format(time.RFC3339Nano))
	TrailarrLog(DEBUG, "Tasks", "StartBackgroundTasks: entering function")
	times, err := LoadTaskTimes()
	if err != nil {
		TrailarrLog(WARN, "Tasks", "Could not load last task times: %v", err)
	}
	TrailarrLog(DEBUG, "Tasks", "Loaded times.Extras: %+v", times.Extras)
	// Get intervals from config (declare once)
	radarrInterval := time.Duration(Timings["radarr"]) * time.Minute
	sonarrInterval := time.Duration(Timings["sonarr"]) * time.Minute
	extrasInterval := time.Duration(Timings["extras"]) * time.Minute

	// If times.Radarr or times.Sonarr or times.Extras is not set, save as now plus interval
	if times.Radarr.TaskId == "" {
		times.Radarr.TaskId = "radarr"
		times.Radarr.Name = "Sync with Radarr"
		times.Radarr.Interval = int(radarrInterval.Minutes())
		saveTaskTimes(times)
		TrailarrLog(DEBUG, "Tasks", "Initialized times.Radarr: %+v", times.Radarr)
	}
	if times.Sonarr.TaskId == "" {
		times.Sonarr.TaskId = "sonarr"
		times.Sonarr.Name = "Sync with Sonarr"
		times.Sonarr.Interval = int(sonarrInterval.Minutes())
		saveTaskTimes(times)
		TrailarrLog(DEBUG, "Tasks", "Initialized times.Sonarr: %+v", times.Sonarr)
	}
	if times.Extras.TaskId == "" {
		times.Extras.TaskId = "extras"
		times.Extras.Name = "Search for Missing Extras"
		times.Extras.Interval = int(extrasInterval.Minutes())
		saveTaskTimes(times)
		TrailarrLog(DEBUG, "Tasks", "Initialized times.Extras: %+v", times.Extras)
	}
	TrailarrLog(DEBUG, "Tasks", "Saving times.Extras: %+v", times.Extras)

	TrailarrLog(DEBUG, "Tasks", "StartBackgroundTasks: initializing scheduler and storage")
	store := storage.NewMemoryStorage()
	sched := scheduler.New(store)

	// Generic persistent scheduling function
	scheduleNext := func(lastRun string, interval time.Duration, taskId string, runJob func(), updateTime func()) {
		var recur func(string)
		recur = func(lastRun string) {
			now := time.Now()
			var nextRun time.Time
			if lastRun != "" {
				last, err := time.Parse(time.RFC3339, lastRun)
				if err == nil {
					nextRun = last.Add(interval)
					if now.After(nextRun) {
						TrailarrLog(INFO, "Tasks", "Missed %s job, running immediately.", taskId)
						runJob()
						updateTime()
						TrailarrLog(INFO, "Tasks", "%s job executed at: %s", taskId, now.Format(time.RFC3339))
						nextRun = now.Add(interval)
					}
				} else {
					nextRun = now.Add(interval)
				}
			} else {
				nextRun = now.Add(interval)
			}
			TrailarrLog(INFO, "Tasks", "Next %s execution: %s", taskId, nextRun.Local().Format("Mon Jan 2 15:04:05 2006 MST"))
			sched.RunAt(nextRun, task.Function(func(params ...task.Param) {
				runJob()
				updateTime()
				TrailarrLog(INFO, "Tasks", "%s job executed at: %s", taskId, time.Now().Format(time.RFC3339))
				recur(time.Now().Format(time.RFC3339))
			}))
		}
		recur(lastRun)
	}

	// Get intervals from config
	// intervals already declared above

	// Helper to check if a task should run immediately
	shouldRunNowTime := func(lastExecution time.Time, interval time.Duration) bool {
		if lastExecution.IsZero() {
			return true
		}
		return lastExecution.Add(interval).Before(time.Now()) || lastExecution.Add(interval).Equal(time.Now())
	}

	// Radarr
	if shouldRunNowTime(times.Radarr.LastExecution, radarrInterval) {
		radarrTaskStarted = true
		broadcastTaskStatus(getCurrentTaskStatus())
		start := time.Now()
		SyncRadarr()
		duration := time.Since(start)
		radarrTaskStarted = false
		broadcastTaskStatus(getCurrentTaskStatus())
		times.Radarr.LastDuration = duration.Seconds()
		times.Radarr.LastExecution = start
		saveTaskTimes(times)
	}
	scheduleNext(times.Radarr.LastExecution.Format(time.RFC3339), radarrInterval, "Radarr",
		func() {
			radarrTaskStarted = true
			broadcastTaskStatus(getCurrentTaskStatus())
			start := time.Now()
			SyncRadarr()
			duration := time.Since(start)
			radarrTaskStarted = false
			broadcastTaskStatus(getCurrentTaskStatus())
			times.Radarr.LastDuration = duration.Seconds()
			times.Radarr.LastExecution = start
			saveTaskTimes(times)
		},
		func() {},
	)

	// Sonarr
	if shouldRunNowTime(times.Sonarr.LastExecution, sonarrInterval) {
		sonarrTaskStarted = true
		broadcastTaskStatus(getCurrentTaskStatus())
		start := time.Now()
		SyncSonarr()
		duration := time.Since(start)
		sonarrTaskStarted = false
		broadcastTaskStatus(getCurrentTaskStatus())
		times.Sonarr.LastDuration = duration.Seconds()
		times.Sonarr.LastExecution = start
		saveTaskTimes(times)
	}
	scheduleNext(times.Sonarr.LastExecution.Format(time.RFC3339), sonarrInterval, "Sonarr",
		func() {
			sonarrTaskStarted = true
			broadcastTaskStatus(getCurrentTaskStatus())
			start := time.Now()
			SyncSonarr()
			duration := time.Since(start)
			sonarrTaskStarted = false
			broadcastTaskStatus(getCurrentTaskStatus())
			times.Sonarr.LastDuration = duration.Seconds()
			times.Sonarr.LastExecution = start
			saveTaskTimes(times)
		},
		func() {},
	)

	// Extras
	if shouldRunNowTime(times.Extras.LastExecution, extrasInterval) {
		TrailarrLog(DEBUG, "Tasks", "[DEBUG] StartBackgroundTasks: calling StartExtrasDownloadTask (no extrasTaskStarted assignment)")
		broadcastTaskStatus(getCurrentTaskStatus())
		start := time.Now()
		StartExtrasDownloadTask()
		duration := time.Since(start)
		broadcastTaskStatus(getCurrentTaskStatus())
		times.Extras.LastDuration = duration.Seconds()
		times.Extras.LastExecution = start
		saveTaskTimes(times)
	}
	TrailarrLog(DEBUG, "Tasks", "StartBackgroundTasks: scheduling Extras task with interval=%v, lastExecution=%v", extrasInterval, times.Extras.LastExecution)
	scheduleNext(times.Extras.LastExecution.Format(time.RFC3339), extrasInterval, "Extras",
		func() {
			TrailarrLog(DEBUG, "Tasks", "StartBackgroundTasks: running scheduled Extras task (no extrasTaskStarted assignment)")
			broadcastTaskStatus(getCurrentTaskStatus())
			start := time.Now()
			StartExtrasDownloadTask()
			duration := time.Since(start)
			broadcastTaskStatus(getCurrentTaskStatus())
			times.Extras.LastDuration = duration.Seconds()
			times.Extras.LastExecution = start
			saveTaskTimes(times)
		},
		func() {},
	)

	// Start the scheduler
	TrailarrLog(DEBUG, "Tasks", "StartBackgroundTasks: starting scheduler goroutine")
	go func() {
		TrailarrLog(DEBUG, "Tasks", "Scheduler goroutine: calling sched.Start()")
		if err := sched.Start(); err != nil {
			TrailarrLog(WARN, "Tasks", "Scheduler failed to start: %v", err)
		} else {
			TrailarrLog(DEBUG, "Tasks", "Scheduler started successfully")
		}
	}()

	TrailarrLog(INFO, "Tasks", "Scheduler started. Jobs will persist last execution times in %s", taskTimesFile)

}

func StartExtrasDownloadTask() {
	TrailarrLog(INFO, "Tasks", "StartExtrasDownloadTask called. PID=%d, time=%s, extrasTaskCancel=%v, extrasTaskStarted=%v", os.Getpid(), time.Now().Format(time.RFC3339Nano), extrasTaskCancel, extrasTaskStarted)
	TrailarrLog(DEBUG, "Tasks", "StartExtrasDownloadTask: entering function")
	TrailarrLog(DEBUG, "Tasks", "[DEBUG] StartExtrasDownloadTask: checking extrasTaskStarted=%v", extrasTaskStarted)
	if extrasTaskCancel != nil || extrasTaskStarted {
		TrailarrLog(WARN, "Tasks", "Attempted to start extras download task, but one is already running. extrasTaskCancel=%v, extrasTaskStarted=%v", extrasTaskCancel, extrasTaskStarted)
		// Cleanup: reset flags so future tasks can start
		if extrasTaskCancel != nil {
			extrasTaskCancel()
			extrasTaskCancel = nil
		}
		TrailarrLog(DEBUG, "Tasks", "[DEBUG] StartExtrasDownloadTask: setting extrasTaskStarted=false (cleanup)")
		extrasTaskStarted = false
		TrailarrLog(INFO, "Tasks", "Extras task flags reset after blocked start.")
		return
	}
	TrailarrLog(DEBUG, "Tasks", "[DEBUG] StartExtrasDownloadTask: setting extrasTaskStarted=true")
	extrasTaskStarted = true
	TrailarrLog(INFO, "Tasks", "Starting extras download task... extrasTaskCancel=%v", extrasTaskCancel)
	ctx, cancel := context.WithCancel(context.Background())
	extrasTaskCancel = cancel
	go func() {
		TrailarrLog(DEBUG, "Tasks", "ExtrasDownloadTask goroutine: started")
		defer func() {
			TrailarrLog(DEBUG, "Tasks", "ExtrasDownloadTask goroutine: exiting, setting extrasTaskStarted=false")
			extrasTaskStarted = false
			TrailarrLog(DEBUG, "Tasks", "[DEBUG] ExtrasDownloadTask goroutine: extrasTaskStarted now=%v", extrasTaskStarted)
		}()
		TrailarrLog(DEBUG, "Tasks", "ExtrasDownloadTask goroutine: calling handleExtrasDownloadLoop, extrasTaskStarted=%v", extrasTaskStarted)
		handleExtrasDownloadLoop(ctx)
		TrailarrLog(DEBUG, "Tasks", "ExtrasDownloadTask goroutine: finished handleExtrasDownloadLoop, exiting goroutine")
	}()
}

func handleExtrasDownloadLoop(ctx context.Context) bool {
	TrailarrLog(DEBUG, "Tasks", "handleExtrasDownloadLoop: entered")
	cfg := mustLoadSearchExtrasConfig()
	interval := getExtrasInterval()
	TrailarrLog(DEBUG, "Tasks", "handleExtrasDownloadLoop: loaded config=%+v, interval=%v", cfg, interval)
	processExtras(ctx, cfg)
	TrailarrLog(DEBUG, "Tasks", "handleExtrasDownloadLoop: processExtras complete, waiting or done")
	result := waitOrDone(ctx, time.Duration(interval)*time.Minute)
	TrailarrLog(DEBUG, "Tasks", "handleExtrasDownloadLoop: waitOrDone returned %v", result)
	return result
}

func waitOrDone(ctx context.Context, d time.Duration) bool {
	select {
	case <-ctx.Done():
		TrailarrLog(INFO, "Tasks", "[TASK] Extras download task stopped by cancel.")
		return true
	case <-time.After(d):
		return false
	}
}

func mustLoadSearchExtrasConfig() SearchExtrasConfig {
	cfg, err := GetSearchExtrasConfig()
	if CheckErrLog(WARN, "Tasks", "Could not load search extras config", err) != nil {
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

func processExtras(ctx context.Context, cfg SearchExtrasConfig) {
	TrailarrLog(DEBUG, "Tasks", "processExtras: entering function with config: %+v", cfg)
	extraTypesCfg, err := GetExtraTypesConfig()
	CheckErrLog(WARN, "Tasks", "Could not load extra types config", err)
	if cfg.SearchMoviesExtras {
		TrailarrLog(INFO, "Tasks", "[TASK] Searching for missing movie extras...")
		DownloadMissingMoviesExtrasWithTypeFilter(ctx, extraTypesCfg)
	} else {
		TrailarrLog(DEBUG, "Tasks", "processExtras: SearchMoviesExtras is disabled")
	}
	if cfg.SearchSeriesExtras {
		TrailarrLog(INFO, "Tasks", "[TASK] Searching for missing series extras...")
		DownloadMissingSeriesExtrasWithTypeFilter(ctx, extraTypesCfg)
	} else {
		TrailarrLog(DEBUG, "Tasks", "processExtras: SearchSeriesExtras is disabled")
	}
}

func StopExtrasDownloadTask() {
	TrailarrLog(DEBUG, "Tasks", "[DEBUG] StopExtrasDownloadTask: checking extrasTaskStarted=%v", extrasTaskStarted)
	if extrasTaskCancel != nil || extrasTaskStarted {
		TrailarrLog(INFO, "Tasks", "Stopping extras download task... extrasTaskCancel=%v, extrasTaskStarted=%v", extrasTaskCancel, extrasTaskStarted)
		if extrasTaskCancel != nil {
			extrasTaskCancel()
			extrasTaskCancel = nil
		}
		TrailarrLog(DEBUG, "Tasks", "[DEBUG] StopExtrasDownloadTask: setting extrasTaskStarted=false")
		extrasTaskStarted = false
	} else {
		TrailarrLog(INFO, "Tasks", "StopExtrasDownloadTask called but extrasTaskCancel is nil and extrasTaskStarted is false")
	}
}

// Download missing movie extras, filtering by enabled types
func DownloadMissingMoviesExtrasWithTypeFilter(ctx context.Context, cfg ExtraTypesConfig) {
	// Example: get all movies, for each, get extras, filter by type, download only enabled types
	downloadMissingExtrasWithTypeFilter(ctx, cfg, "movie", TrailarrRoot+"/movies_wanted.json")
}

// Download missing series extras, filtering by enabled types
func DownloadMissingSeriesExtrasWithTypeFilter(ctx context.Context, cfg ExtraTypesConfig) {
	downloadMissingExtrasWithTypeFilter(ctx, cfg, "tv", TrailarrRoot+"/series_wanted.json")
}

// Shared logic for type-filtered extras download
func downloadMissingExtrasWithTypeFilter(ctx context.Context, cfg ExtraTypesConfig, mediaType MediaType, cacheFile string) {
	TrailarrLog(DEBUG, "Tasks", "Starting downloadMissingExtrasWithTypeFilter: mediaType=%v, cacheFile=%s", mediaType, cacheFile)
	items, err := loadCache(cacheFile)
	if CheckErrLog(DEBUG, "Tasks", "Failed to load cache", err) != nil {
		TrailarrLog(DEBUG, "Tasks", "No items loaded from cache: %s", cacheFile)
		return
	}
	TrailarrLog(DEBUG, "Tasks", "Loaded %d items from cache", len(items))

	for _, item := range items {
		if ctx != nil && ctx.Err() != nil {
			TrailarrLog(INFO, "Tasks", "Extras download cancelled before processing item.")
			break
		}
		mediaId, ok := parseMediaID(item["id"])
		if !ok {
			TrailarrLog(DEBUG, "Tasks", "Skipping item with invalid mediaId: %+v", item)
			continue
		}
		TrailarrLog(DEBUG, "Tasks", "Processing mediaId=%v", mediaId)
		extras, err := SearchExtras(mediaType, mediaId)
		if err != nil {
			TrailarrLog(WARN, "Tasks", "SearchExtras failed for mediaId=%v: %v", mediaId, err)
			continue
		}
		TrailarrLog(DEBUG, "Tasks", "Found %d extras for mediaId=%v", len(extras), mediaId)
		mediaPath, err := FindMediaPathByID(cacheFile, mediaId)
		if err != nil || mediaPath == "" {
			TrailarrLog(WARN, "Tasks", "FindMediaPathByID failed for mediaId=%v: %v", mediaId, err)
			continue
		}
		TrailarrLog(DEBUG, "Tasks", "Media path for mediaId=%v: %s", mediaId, mediaPath)
		MarkDownloadedExtras(extras, mediaPath, "type", "title")
		// For each extra, download sequentially
		for _, extra := range extras {
			if ctx != nil && ctx.Err() != nil {
				TrailarrLog(INFO, "Tasks", "Extras download cancelled before processing extra.")
				break
			}
			typ := canonicalizeExtraType(extra.Type, extra.Type)
			if !isExtraTypeEnabled(cfg, typ) {
				continue
			}
			if extra.Status == "missing" && extra.YoutubeId != "" {
				TrailarrLog(DEBUG, "Tasks", "[SEQ] Downloading extra: mediaType=%v, mediaId=%v, type=%s, title=%s, youtubeId=%s", mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
				err := handleTypeFilteredExtraDownload(mediaType, mediaId, extra)
				if err != nil {
					TrailarrLog(WARN, "Tasks", "[SEQ] Download failed: %v", err)
				}
			}
		}
	}
	TrailarrLog(DEBUG, "Tasks", "All downloads finished.")
}

func handleTypeFilteredExtraDownload(mediaType MediaType, mediaId int, extra Extra) error {
	TrailarrLog(DEBUG, "Tasks", "Downloading YouTube extra: mediaType=%v, mediaId=%v, type=%s, title=%s, youtubeId=%s", mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
	_, err := DownloadYouTubeExtra(mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
	if err != nil {
		TrailarrLog(WARN, "Tasks", "DownloadYouTubeExtra failed: %v", err)
	}
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
	case "Other":
		return cfg.Other
	default:
		return false
	}
}
