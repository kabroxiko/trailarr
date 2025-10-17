package internal

import (
	"bytes"
	"encoding/json"
	"fmt"
	iofs "io/fs"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"trailarr/assets"

	"github.com/gin-gonic/gin"
)

const indexHTMLFilename = "index.html"

func RegisterRoutes(r *gin.Engine) {
	// Register grouped routes to keep this function small
	registerCastRoutes(r)
	registerYouTubeAndProxyRoutes(r)
	registerDownloadAndBlacklistRoutes(r)
	registerTaskWebSocketRoutes(r)
	registerLogAndTMDBRoutes(r)
	_ = EnsureYtdlpFlagsConfigExists()
	registerYtdlpRoutes(r)
	registerAPILogMiddleware(r)

	registerProviderAndTestRoutes(r)
	registerHealthAndTaskRoutes(r)

	// Serve embedded or filesystem static assets
	var distFS iofs.FS
	if s, err := iofs.Sub(assets.EmbeddedDist, "dist"); err == nil {
		distFS = s
		registerEmbeddedStaticRoutes(r, distFS)
	} else {
		// fallback to filesystem if embed not available
		r.Static("/assets", "./web/dist/assets")
		r.StaticFile("/", "./web/dist/index.html")
	}

	registerLogFileRoute(r)
	registerNoRouteHandler(r, distFS)

	// Static media and logo
	r.Static("/mediacover", MediaCoverPath)
	registerLogoRoute(r, distFS)

	registerMediaAndSettingsRoutes(r)

	// Extras and history endpoints
	r.POST("/api/extras/download", downloadExtraHandler)
	r.DELETE("/api/extras", deleteExtraHandler)
	r.GET("/api/extras/existing", existingExtrasHandler)
	r.GET("/api/history", historyHandler)

	// Extra types and canonicalize config endpoints
	r.GET("/api/settings/extratypes", GetExtraTypesConfigHandler)
	r.POST("/api/settings/extratypes", SaveExtraTypesConfigHandler)
	r.GET("/api/settings/canonicalizeextratype", GetCanonicalizeExtraTypeConfigHandler)
	r.POST("/api/settings/canonicalizeextratype", SaveCanonicalizeExtraTypeConfigHandler)

	// TMDB extra types endpoint
	r.GET("/api/tmdb/extratypes", func(c *gin.Context) {
		respondJSON(c, http.StatusOK, gin.H{"tmdbExtraTypes": TMDBExtraTypes})
	})

	// Server-side file browser for directory picker
	r.GET("/api/files/list", ListServerFoldersHandler)
}

func registerCastRoutes(r *gin.Engine) {
	r.GET("/api/movies/:id/cast", func(c *gin.Context) {
		idStr := c.Param("id")
		var id int
		fmt.Sscanf(idStr, "%d", &id)
		tmdbKey, err := GetTMDBKey()
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		tmdbId, err := GetTMDBId(MediaTypeMovie, id)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		cast, err := FetchTMDBCast(MediaTypeMovie, tmdbId, tmdbKey)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(c, http.StatusOK, gin.H{"cast": cast})
	})
	r.GET("/api/series/:id/cast", func(c *gin.Context) {
		idStr := c.Param("id")
		var id int
		fmt.Sscanf(idStr, "%d", &id)
		tmdbKey, err := GetTMDBKey()
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		tmdbId, err := GetTMDBId(MediaTypeTV, id)
		if err != nil {
			respondError(c, http.StatusInternalServerError,
				err.Error())
			return
		}
		cast, err := FetchTMDBCast(MediaTypeTV, tmdbId, tmdbKey)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(c, http.StatusOK, gin.H{"cast": cast})
	})
}

func registerYouTubeAndProxyRoutes(r *gin.Engine) {
	r.POST("/api/youtube/search", YouTubeTrailerSearchHandler)
	r.GET("/api/youtube/search/stream", YouTubeTrailerSearchStreamHandler)
	r.GET("/api/proxy/youtube-image/:youtubeId", ProxyYouTubeImageHandler)
	r.HEAD("/api/proxy/youtube-image/:youtubeId", ProxyYouTubeImageHandler)
}

func registerDownloadAndBlacklistRoutes(r *gin.Engine) {
	// WebSocket for real-time download queue updates
	r.GET("/ws/download-queue", func(c *gin.Context) {
		wsUpgrader := getWebSocketUpgrader()
		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			TrailarrLog(WARN, "WS", "WebSocket upgrade failed: %v", err)
			return
		}
		AddDownloadQueueClient(conn)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
		RemoveDownloadQueueClient(conn)
		conn.Close()
	})
	// Download status endpoints
	r.GET("/api/extras/status/:youtubeId", GetDownloadStatusHandler)
	r.POST("/api/extras/status/batch", GetBatchDownloadStatusHandler)
	// Start the download queue worker
	StartDownloadQueueWorker()
	r.GET("/api/blacklist/extras", BlacklistExtrasHandler)
	r.POST("/api/blacklist/extras/remove", RemoveBlacklistExtraHandler)
}

func registerTaskWebSocketRoutes(r *gin.Engine) {
	// WebSocket for real-time task status
	r.GET("/ws/tasks", func(c *gin.Context) {
		wsUpgrader := getWebSocketUpgrader()
		conn, err := wsUpgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			TrailarrLog(WARN, "WS", "WebSocket upgrade failed: %v", err)
			return
		}
		addTaskStatusClient(conn)
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				break
			}
		}
		removeTaskStatusClient(conn)
		conn.Close()
	})
}

func registerLogAndTMDBRoutes(r *gin.Engine) {
	r.GET("/api/logs/list", logsListHandler)
	r.GET("/api/test/tmdb", testTMDBHandler)
}

func logsListHandler(c *gin.Context) {
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
}

func testTMDBHandler(c *gin.Context) {
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
		return
	}
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

func registerYtdlpRoutes(r *gin.Engine) {
	r.GET("/api/settings/ytdlpflags", GetYtdlpFlagsConfigHandler)
	r.POST("/api/settings/ytdlpflags", SaveYtdlpFlagsConfigHandler)
}

func registerAPILogMiddleware(r *gin.Engine) {
	// Log all API calls except /mediacover
	r.Use(func(c *gin.Context) {
		// Omit logging for /mediacover and GET /api/tasks/queue
		if !(c.Request.Method == "GET" && c.Request.URL.Path == "/api/tasks/queue") &&
			(len(c.Request.URL.Path) < 11 || c.Request.URL.Path[:11] != "/mediacover") {
			TrailarrLog(INFO, "API", "%s %s", c.Request.Method, c.Request.URL.Path)
		}
		c.Next()
	})
}

func registerProviderAndTestRoutes(r *gin.Engine) {
	r.GET("/api/rootfolders", func(c *gin.Context) {
		providerURL := c.Query("providerURL")
		apiKey := c.Query("apiKey")
		if providerURL == "" || apiKey == "" {
			respondError(c, http.StatusBadRequest, "Missing providerURL or apiKey")
			return
		}
		folders, err := FetchRootFolders(providerURL, apiKey)
		if err != nil {
			respondError(c, http.StatusInternalServerError, err.Error())
			return
		}
		respondJSON(c, http.StatusOK, folders)
	})

	r.GET("/api/test/:provider", func(c *gin.Context) {
		provider := c.Param("provider")
		url := c.Query("url")
		apiKey := c.Query("apiKey")
		if url == "" || apiKey == "" {
			respondError(c, http.StatusBadRequest,
				"Missing url or apiKey")
			return
		}
		err := testMediaConnection(url, apiKey, provider)
		if err != nil {
			respondError(c, http.StatusOK, err.Error())
		} else {
			respondJSON(c, http.StatusOK, gin.H{"success": true})
		}
	})
}

func registerHealthAndTaskRoutes(r *gin.Engine) {
	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		respondJSON(c, http.StatusOK, gin.H{"status": "ok"})
	})

	// API endpoint for scheduled/queue status
	r.GET("/api/tasks/status", GetAllTasksStatus())
	r.GET("/api/tasks/queue", GetTaskQueueFileHandler())
	r.POST("/api/tasks/force", TaskHandler())
}

func registerEmbeddedStaticRoutes(r *gin.Engine, distFS iofs.FS) {
	// serve assets from embedded dist/assets
	r.GET("/assets/*filepath", func(c *gin.Context) {
		p := c.Param("filepath")
		// attempt to open file first
		if _, err := distFS.Open(filepath.Join("assets", p)); err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		buf, err := iofs.ReadFile(distFS, filepath.Join("assets", p))
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		reader := bytes.NewReader(buf)
		http.ServeContent(c.Writer, c.Request, p, time.Now(), reader)
	})
	// serve index.html at root
	r.GET("/", func(c *gin.Context) {
		data, err := iofs.ReadFile(distFS, indexHTMLFilename)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}
		reader := bytes.NewReader(data)
		http.ServeContent(c.Writer, c.Request, indexHTMLFilename, time.Now(), reader)
	})
}

func registerLogFileRoute(r *gin.Engine) {
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
}

func registerNoRouteHandler(r *gin.Engine, distFS iofs.FS) {
	r.NoRoute(func(c *gin.Context) {
		TrailarrLog(INFO, "WEB", "NoRoute handler hit for path: %s", c.Request.URL.Path)
		// Serve index.html from embed if possible
		if distFS != nil {
			data, err := iofs.ReadFile(distFS, indexHTMLFilename)
			if err == nil {
				reader := bytes.NewReader(data)
				http.ServeContent(c.Writer, c.Request, indexHTMLFilename, time.Now(), reader)
				return
			}
		}
		c.File("./web/dist/index.html")
	})
}

func registerLogoRoute(r *gin.Engine, distFS iofs.FS) {
	r.GET("/logo.svg", func(c *gin.Context) {
		if distFS != nil {
			if data, err := iofs.ReadFile(distFS, "logo.svg"); err == nil {
				reader := bytes.NewReader(data)
				http.ServeContent(c.Writer, c.Request, "logo.svg", time.Now(), reader)
				return
			}
		}
		c.File("web/public/logo.svg")
	})
}

func registerMediaAndSettingsRoutes(r *gin.Engine) {
	// Helper for default media path
	// Group movies/series endpoints
	for _, media := range []struct {
		section      string
		cacheFile    string
		fallbackPath string
		extrasType   MediaType
	}{
		{"movies", MoviesRedisKey,
			"/Movies", MediaTypeMovie},
		{"series", SeriesRedisKey, "/Series", MediaTypeTV},
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
	// General settings (TMDB key)
	r.GET("/api/settings/general", getGeneralSettingsHandler)
	r.POST("/api/settings/general", saveGeneralSettingsHandler)
}
