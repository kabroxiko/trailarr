package internal

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Download status by YouTube ID
	r.GET("/api/extras/status/:youtubeId", GetDownloadStatusHandler)
	// Batch status endpoint
	r.POST("/api/extras/status/batch", GetBatchDownloadStatusHandler)
	// Start the download queue worker
	StartDownloadQueueWorker()
	r.GET("/api/blacklist/extras", BlacklistExtrasHandler)
	r.POST("/api/blacklist/extras/remove", RemoveBlacklistExtraHandler)
	// --- WebSocket for real-time task status ---
	r.GET("/ws/tasks", func(c *gin.Context) {
		// Upgrade connection to WebSocket
		wsUpgrader := getWebSocketUpgrader()
		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			TrailarrLog(WARN, "WS", "WebSocket upgrade failed: %v", err)
			return
		}
		addTaskStatusClient(conn)
		// Keep connection open until closed by client
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
		removeTaskStatusClient(conn)
		conn.Close()
	})
	r.GET("/api/logs/list", func(c *gin.Context) {
		entries, err := filepath.Glob(LogsDir + "/*.txt")
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		type LogInfo struct {
			Number    int    `json:"number"`
			Filename  string `json:"filename"`
			LastWrite string `json:"lastWrite"`
		}
		var logs []LogInfo
		for i, path := range entries {
			fi, err := os.Stat(path)
			if err != nil {
				continue
			}
			logs = append(logs, LogInfo{
				Number:    i + 1,
				Filename:  filepath.Base(path),
				LastWrite: fi.ModTime().Format("02 Jan 2006 15:04"),
			})
		}
		respondJSON(c, http.StatusOK, gin.H{"logs": logs, "logDir": LogsDir})
	})
	// Test TMDB API key endpoint
	r.GET("/api/test/tmdb", func(c *gin.Context) {
		apiKey := c.Query("apiKey")
		if apiKey == "" {
			respondError(c, http.StatusBadRequest, "Missing apiKey")
			return
		}
		testUrl := "https://api.themoviedb.org/3/configuration?api_key=" + apiKey
		resp, err := http.Get(testUrl)
		if err != nil {
			respondError(c, http.StatusOK, err.Error())
			return
		}
		defer resp.Body.Close()
		if resp.StatusCode == 200 {
			respondJSON(c, http.StatusOK, gin.H{"success": true})
		} else {
			var body struct {
				StatusMessage string `json:"status_message"`
			}
			_ = json.NewDecoder(resp.Body).Decode(&body)
			msg := body.StatusMessage
			if msg == "" {
				msg = "Invalid TMDB API Key"
			}
			respondError(c, http.StatusOK, msg)
		}
	})
	// Ensure yt-dlp config exists at startup
	_ = EnsureYtdlpFlagsConfigExists()
	// YTDLP flags config endpoints
	r.GET("/api/settings/ytdlpflags", GetYtdlpFlagsConfigHandler)
	r.POST("/api/settings/ytdlpflags", SaveYtdlpFlagsConfigHandler)
	// Log all API calls except /mediacover
	r.Use(func(c *gin.Context) {
		// Omit logging for /mediacover and GET /api/tasks/queue
		if !(c.Request.Method == "GET" && c.Request.URL.Path == "/api/tasks/queue") &&
			(len(c.Request.URL.Path) < 11 || c.Request.URL.Path[:11] != "/mediacover") {
			TrailarrLog(INFO, "API", "%s %s", c.Request.Method, c.Request.URL.Path)
		}
		c.Next()
	})

	// Endpoint to fetch root folders for Radarr/Sonarr
	r.GET("/api/rootfolders", func(c *gin.Context) {
		url := c.Query("url")
		apiKey := c.Query("apiKey")
		if url == "" || apiKey == "" {
			respondError(c, http.StatusBadRequest, "Missing url or apiKey")
			return
		}
		folders, err := FetchRootFolders(url, apiKey)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(c, http.StatusOK, folders)
	})
	// Combined test connection endpoint for Radarr/Sonarr
	r.GET("/api/test/:provider", func(c *gin.Context) {
		provider := c.Param("provider")
		url := c.Query("url")
		apiKey := c.Query("apiKey")
		if url == "" || apiKey == "" {
			respondError(c, http.StatusBadRequest, "Missing url or apiKey")
			return
		}
		err := testMediaConnection(url, apiKey, provider)
		if err != nil {
			respondError(c, http.StatusOK, err.Error())
		} else {
			respondJSON(c, http.StatusOK, gin.H{"success": true})
		}
	})
	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		respondJSON(c, http.StatusOK, gin.H{"status": "ok"})
	})

	// API endpoint for scheduled/queue status
	r.GET("/api/tasks/status", GetAllTasksStatus())
	r.GET("/api/tasks/queue", GetTaskQueueFileHandler())
	r.POST("/api/tasks/force", TaskHandler())

	// Serve React static files and SPA fallback
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")

	// Serve log files for frontend log viewer
	r.GET("/logs/:filename", func(c *gin.Context) {
		filename := c.Param("filename")
		filePath := LogsDir + "/" + filename
		// Security: only allow .txt files and prevent path traversal
		if len(filename) < 5 || filename[len(filename)-4:] != ".txt" || filename != filepath.Base(filename) {
			respondError(c, http.StatusBadRequest, "Invalid log filename")
			return
		}
		c.File(filePath)
	})

	r.NoRoute(func(c *gin.Context) {
		TrailarrLog(INFO, "WEB", "NoRoute handler hit for path: %s", c.Request.URL.Path)
		c.File("./web/dist/index.html")
	})

	// Serve static files for movie posters
	r.Static("/mediacover", MediaCoverPath)
	r.StaticFile("/logo.svg", "web/public/logo.svg")
	// Helper for default media path
	// Group movies/series endpoints
	for _, media := range []struct {
		section      string
		cacheFile    string
		fallbackPath string
		extrasType   MediaType
	}{
		{"movies", MoviesJSONPath, "/Movies", MediaTypeMovie},
		{"series", SeriesJSONPath, "/Series", MediaTypeTV},
	} {
		r.GET("/api/"+media.section, GetMediaHandler(media.cacheFile, "id"))
		r.GET("/api/"+media.section+"/wanted", GetMissingExtrasHandler(media.cacheFile))
		r.GET("/api/"+media.section+"/:id", GetMediaByIdHandler(media.cacheFile, "id"))
		r.GET("/api/"+media.section+"/:id/extras", sharedExtrasHandler(media.extrasType))
	}
	// Group settings endpoints for Radarr/Sonarr
	for _, provider := range []string{"radarr", "sonarr"} {
		r.GET("/api/settings/"+provider, GetSettingsHandler(provider))
		r.POST("/api/settings/"+provider, SaveSettingsHandler(provider))
	}
	r.POST("/api/extras/download", downloadExtraHandler)
	r.DELETE("/api/extras", deleteExtraHandler)
	r.GET("/api/extras/existing", existingExtrasHandler)
	r.GET("/api/history", historyHandler)
	// Posters and banners are now served directly from /mediacover static path
	// General settings (TMDB key)
	r.GET("/api/settings/general", getGeneralSettingsHandler)
	r.POST("/api/settings/general", saveGeneralSettingsHandler)

	// Extra types config endpoints
	r.GET("/api/settings/extratypes", GetExtraTypesConfigHandler)
	r.POST("/api/settings/extratypes", SaveExtraTypesConfigHandler)

	// CanonicalizeExtraType config endpoints
	r.GET("/api/settings/canonicalizeextratype", GetCanonicalizeExtraTypeConfigHandler)
	r.POST("/api/settings/canonicalizeextratype", SaveCanonicalizeExtraTypeConfigHandler)

	// TMDB extra types endpoint
	r.GET("/api/tmdb/extratypes", func(c *gin.Context) {
		respondJSON(c, http.StatusOK, gin.H{"tmdbExtraTypes": TMDBExtraTypes})
	})

	// Server-side file browser for directory picker
	r.GET("/api/files/list", ListServerFoldersHandler)
}
