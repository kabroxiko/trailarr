package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	scheduler "github.com/algorythma/go-scheduler"
	"github.com/algorythma/go-scheduler/storage"
	"github.com/algorythma/go-scheduler/task"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

var taskStatusClientsMu sync.Mutex
var taskStatusClients = make(map[*websocket.Conn]struct{})

// Broadcasts the full status of all tasks, ignoring partial input
func broadcastTaskStatus(_ map[string]interface{}) {
	// Always send the current status of all tasks
	status := getCurrentTaskStatus()
	taskStatusClientsMu.Lock()
	numClients := len(taskStatusClients)
	taskStatusClientsMu.Unlock()
	TrailarrLog(DEBUG, "Tasks", "[BROADCAST] Sending to %d clients: %v", numClients, status)
	// Optionally, send to all clients for debug
	for conn := range taskStatusClients {
		sendTaskStatus(conn, status)
	}
}

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

// TaskTimes is now a map of tasks
type TaskTimes map[string]Task

// Static map of tasks with id, name, and interval
type TaskMeta struct {
	TaskId   string
	Name     string
	Interval int
}

var tasks = map[string]TaskMeta{
	"radarr": {
		TaskId:   "radarr",
		Name:     "Sync with Radarr",
		Interval: 0, // will be set from config
	},
	"sonarr": {
		TaskId:   "sonarr",
		Name:     "Sync with Sonarr",
		Interval: 0, // will be set from config
	},
	"extras": {
		TaskId:   "extras",
		Name:     "Search for Missing Extras",
		Interval: 0, // will be set from config
	},
}

func LoadTaskTimes() (TaskTimes, error) {
	var arr []Task
	path := filepath.Join(TrailarrRoot, taskTimesFile)
	err := ReadJSONFile(path, &arr)
	times := make(map[string]Task)
	if err != nil {
		// If file does not exist, initialize with empty times and create file
		if os.IsNotExist(err) {
			times["radarr"] = Task{TaskId: "radarr", Name: tasks["radarr"].Name, Interval: Timings["radarr"]}
			times["sonarr"] = Task{TaskId: "sonarr", Name: tasks["sonarr"].Name, Interval: Timings["sonarr"]}
			times["extras"] = Task{TaskId: "extras", Name: tasks["extras"].Name, Interval: Timings["extras"]}
			_ = saveTaskTimes(times)
			GlobalTaskTimes = times
			return times, nil
		}
		return times, err
	}
	// Map array entries to map by TaskId
	for _, t := range arr {
		// Always set Name from static map
		t.Name = tasks[t.TaskId].Name
		t.Interval = Timings[t.TaskId]
		times[t.TaskId] = t
	}
	GlobalTaskTimes = times
	return times, nil
}

func saveTaskTimes(times map[string]Task) error {
	GlobalTaskTimes = times
	arr := make([]Task, 0, len(times))
	for _, t := range times {
		arr = append(arr, t)
	}
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
		schedules := buildSchedules(times, radarrStatus, sonarrStatus, extrasStatus)
		queues := buildTaskQueues()
		sortTaskQueuesByQueuedDesc(queues)
		respondJSON(c, http.StatusOK, gin.H{
			"schedules": schedules,
			"queues":    queues,
		})
	}
}

// Helper to calculate next execution time
func calcNext(lastExecution time.Time, interval int) time.Time {
	if lastExecution.IsZero() {
		return time.Now().Add(time.Duration(interval) * time.Minute)
	}
	return lastExecution.Add(time.Duration(interval) * time.Minute)
}

// Helper to build schedules array
func buildSchedules(times map[string]Task, radarrStatus, sonarrStatus, extrasStatus string) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"taskId":        times["radarr"].TaskId,
			"name":          times["radarr"].Name,
			"interval":      times["radarr"].Interval,
			"lastExecution": times["radarr"].LastExecution,
			"lastDuration":  times["radarr"].LastDuration,
			"nextExecution": calcNext(times["radarr"].LastExecution, times["radarr"].Interval),
			"status":        radarrStatus,
		},
		{
			"taskId":        times["sonarr"].TaskId,
			"name":          times["sonarr"].Name,
			"interval":      times["sonarr"].Interval,
			"lastExecution": times["sonarr"].LastExecution,
			"lastDuration":  times["sonarr"].LastDuration,
			"nextExecution": calcNext(times["sonarr"].LastExecution, times["sonarr"].Interval),
			"status":        sonarrStatus,
		},
		{
			"taskId":        times["extras"].TaskId,
			"name":          times["extras"].Name,
			"interval":      times["extras"].Interval,
			"lastExecution": times["extras"].LastExecution,
			"lastDuration":  times["extras"].LastDuration,
			"nextExecution": calcNext(times["extras"].LastExecution, times["extras"].Interval),
			"status":        extrasStatus,
		},
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
		"TaskId":   item.TaskId,
		"Queued":   item.Queued,
		"Started":  getTimeOrEmpty(item.Started),
		"Ended":    getTimeOrEmpty(item.Ended),
		"Duration": getDurationOrEmpty(item.Duration),
		"Status":   item.Status,
		"Error":    item.Error,
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
	type forceTask struct {
		id       string
		started  *bool
		syncFunc func()
		respond  string
	}
	tasks := map[string]forceTask{
		"radarr": {"radarr", &radarrTaskStarted, SyncRadarr, "Sync Radarr forced"},
		"sonarr": {"sonarr", &sonarrTaskStarted, SyncSonarr, "Sync Sonarr forced"},
		"extras": {"extras", nil, StartExtrasDownloadTask, "Sync Extras forced"},
	}
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
		t, ok := tasks[req.TaskId]
		if !ok {
			TrailarrLog(DEBUG, "Tasks", "[FORCE] Unknown job requested: %s", req.TaskId)
			respondError(c, http.StatusBadRequest, "unknown task")
			return
		}
		TrailarrLog(DEBUG, "Tasks", "[FORCE] Starting %s job at %s", t.id, time.Now().Format(time.RFC3339Nano))
		if t.started != nil {
			*t.started = true
		}
		broadcastTaskStatus(map[string]interface{}{
			"schedules": []map[string]interface{}{
				{
					"taskId":        times[t.id].TaskId,
					"name":          times[t.id].Name,
					"interval":      times[t.id].Interval,
					"lastExecution": times[t.id].LastExecution,
					"lastDuration":  times[t.id].LastDuration,
					"nextExecution": calcNext(times[t.id].LastExecution, times[t.id].Interval),
					"status":        "running",
				},
			},
		})
		start := time.Now()
		t.syncFunc()
		duration := time.Since(start)
		if t.started != nil {
			*t.started = false
		}
		broadcastTaskStatus(getCurrentTaskStatus())
		TrailarrLog(DEBUG, "Tasks", "[FORCE] %s job finished at %s, duration=%s", t.id, time.Now().Format(time.RFC3339Nano), duration.String())
		taskTimes := times[t.id]
		taskTimes.LastDuration = duration.Seconds()
		taskTimes.LastExecution = start
		times[t.id] = taskTimes
		saveTaskTimes(times)
		TrailarrLog(DEBUG, "Tasks", "[FORCE] Updated times.%s: %+v", t.id, times[t.id])
		respondJSON(c, http.StatusOK, gin.H{"status": t.respond})
	}
}

func StartBackgroundTasks() {
	TrailarrLog(INFO, "Tasks", "StartBackgroundTasks called. PID=%d, time=%s", os.Getpid(), time.Now().Format(time.RFC3339Nano))
	TrailarrLog(DEBUG, "Tasks", "StartBackgroundTasks: entering function")
	times, err := LoadTaskTimes()
	if err != nil {
		TrailarrLog(WARN, "Tasks", "Could not load last task times: %v", err)
	}
	TrailarrLog(DEBUG, "Tasks", "Loaded times.Extras: %+v", times["extras"])

	type bgTask struct {
		id        string
		started   *bool
		syncFunc  func()
		interval  time.Duration
		lastExec  time.Time
		logPrefix string
	}
	taskList := []bgTask{
		{"radarr", &radarrTaskStarted, SyncRadarr, time.Duration(Timings["radarr"]) * time.Minute, times["radarr"].LastExecution, "Radarr"},
		{"sonarr", &sonarrTaskStarted, SyncSonarr, time.Duration(Timings["sonarr"]) * time.Minute, times["sonarr"].LastExecution, "Sonarr"},
		{"extras", &extrasTaskStarted, StartExtrasDownloadTask, time.Duration(Timings["extras"]) * time.Minute, times["extras"].LastExecution, "Extras"},
	}

	TrailarrLog(DEBUG, "Tasks", "StartBackgroundTasks: initializing scheduler and storage")
	store := storage.NewMemoryStorage()
	sched := scheduler.New(store)

	scheduleNext := func(lastRun string, interval time.Duration, taskId string, runJob func()) {
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
				TrailarrLog(INFO, "Tasks", "%s job executed at: %s", taskId, time.Now().Format(time.RFC3339))
				recur(time.Now().Format(time.RFC3339))
			}))
		}
		recur(lastRun)
	}

	shouldRunNowTime := func(lastExecution time.Time, interval time.Duration) bool {
		if lastExecution.IsZero() {
			return true
		}
		return lastExecution.Add(interval).Before(time.Now()) || lastExecution.Add(interval).Equal(time.Now())
	}

	for i := range taskList {
		t := &taskList[i]
		if shouldRunNowTime(t.lastExec, t.interval) {
			if t.started != nil {
				*t.started = true
			}
			broadcastTaskStatus(getCurrentTaskStatus())
			start := time.Now()
			t.syncFunc()
			duration := time.Since(start)
			if t.started != nil {
				*t.started = false
			}
			broadcastTaskStatus(getCurrentTaskStatus())
			taskTimes := times[t.id]
			taskTimes.LastDuration = duration.Seconds()
			taskTimes.LastExecution = start
			times[t.id] = taskTimes
			saveTaskTimes(times)
		}
		TrailarrLog(DEBUG, "Tasks", "StartBackgroundTasks: scheduling %s task with interval=%v, lastExecution=%v", t.logPrefix, t.interval, t.lastExec)
		scheduleNext(t.lastExec.Format(time.RFC3339), t.interval, t.logPrefix, func() {
			if t.started != nil {
				*t.started = true
			}
			broadcastTaskStatus(getCurrentTaskStatus())
			start := time.Now()
			t.syncFunc()
			duration := time.Since(start)
			if t.started != nil {
				*t.started = false
			}
			broadcastTaskStatus(getCurrentTaskStatus())
			taskTimes := times[t.id]
			taskTimes.LastDuration = duration.Seconds()
			taskTimes.LastExecution = start
			times[t.id] = taskTimes
			saveTaskTimes(times)
		})
	}

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
		// Defensive: mark rejected extras before any download
		rejectedExtras := GetRejectedExtrasForMedia(mediaType, mediaId)
		rejectedYoutubeIds := make(map[string]struct{})
		for _, r := range rejectedExtras {
			rejectedYoutubeIds[r.YoutubeId] = struct{}{}
		}
		for i := range extras {
			if _, exists := rejectedYoutubeIds[extras[i].YoutubeId]; exists {
				extras[i].Status = "rejected"
			}
		}
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
			if extra.Status == "rejected" {
				TrailarrLog(DEBUG, "Tasks", "[SEQ] Skipping rejected extra: mediaType=%v, mediaId=%v, type=%s, title=%s, youtubeId=%s", mediaType, mediaId, extra.Type, extra.Title, extra.YoutubeId)
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
		return err
	}
	// Add to history if download succeeded
	AppendHistoryEvent(HistoryEvent{
		Action:     "download",
		Title:      getMediaTitleFromCache(mediaType, mediaId),
		MediaType:  mediaType,
		MediaId:    mediaId,
		ExtraType:  extra.Type,
		ExtraTitle: extra.Title,
		Date:       time.Now(),
	})
	return nil
}

// Helper to get media title from cache
func getMediaTitleFromCache(mediaType MediaType, mediaId int) string {
	cacheFile, _ := resolveCachePath(mediaType)
	if cacheFile != "" {
		items, _ := loadCache(cacheFile)
		for _, m := range items {
			idInt, ok := parseMediaID(m["id"])
			if ok && idInt == mediaId {
				if t, ok := m["title"].(string); ok {
					return t
				}
			}
		}
	}
	return ""
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

func getWebSocketUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
}

func addTaskStatusClient(conn *websocket.Conn) {
	taskStatusClientsMu.Lock()
	taskStatusClients[conn] = struct{}{}
	taskStatusClientsMu.Unlock()
	// Send initial status
	go sendCurrentTaskStatus(conn)
}

func removeTaskStatusClient(conn *websocket.Conn) {
	taskStatusClientsMu.Lock()
	delete(taskStatusClients, conn)
	taskStatusClientsMu.Unlock()
}

func sendCurrentTaskStatus(conn *websocket.Conn) {
	status := getCurrentTaskStatus()
	sendTaskStatus(conn, status)
}

func sendTaskStatus(conn *websocket.Conn, status interface{}) {
	data, err := json.Marshal(status)
	if err != nil {
		return
	}
	conn.WriteMessage(websocket.TextMessage, data)
}

// Returns a map with all tasks' current status for broadcasting
func getCurrentTaskStatus() map[string]interface{} {
	times := GlobalTaskTimes
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
	return map[string]interface{}{
		"schedules": buildSchedules(times, radarrStatus, sonarrStatus, extrasStatus),
	}
}

// Generic handler for Radarr/Sonarr sync status
func GetSyncStatusHandler(section string, status *SyncStatus, displayName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		interval := Timings[section]
		respondJSON(c, http.StatusOK, gin.H{
			"scheduled": gin.H{
				"name":          "Sync with " + displayName,
				"interval":      fmt.Sprintf("%d minutes", interval),
				"lastExecution": LastExecution(status),
				"lastDuration":  LastDuration(status).String(),
				"nextExecution": NextExecution(status),
				"lastError":     LastError(status),
			},
			"queue": Queue(status),
		})
	}
}
