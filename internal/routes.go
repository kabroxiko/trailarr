package internal

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Ensure yt-dlp config exists at startup
	_ = EnsureYtdlpFlagsConfigExists()
	// YTDLP flags config endpoints
	r.GET("/api/settings/ytdlpflags", GetYtdlpFlagsConfigHandler)
	r.POST("/api/settings/ytdlpflags", SaveYtdlpFlagsConfigHandler)
	// Log all API calls except /mediacover
	r.Use(func(c *gin.Context) {
		if len(c.Request.URL.Path) < 11 || c.Request.URL.Path[:11] != "/mediacover" {
			TrailarrLog("Info", "API", "%s %s", c.Request.Method, c.Request.URL.Path)
		}
		c.Next()
	})

	// Endpoint to fetch root folders for Radarr/Sonarr
	r.GET("/api/rootfolders", func(c *gin.Context) {
		url := c.Query("url")
		apiKey := c.Query("apiKey")
		if url == "" || apiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Missing url or apiKey"})
			return
		}
		folders, err := FetchRootFolders(url, apiKey)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error(), "folders": []string{}})
			return
		}
		c.JSON(http.StatusOK, folders)
	})
	// Test connection endpoints for Radarr/Sonarr
	r.GET("/api/test/radarr", func(c *gin.Context) {
		url := c.Query("url")
		apiKey := c.Query("apiKey")
		if url == "" || apiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing url or apiKey"})
			return
		}
		err := testMediaConnection(url, apiKey, "radarr")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"success": true})
		}
	})
	r.GET("/api/test/sonarr", func(c *gin.Context) {
		url := c.Query("url")
		apiKey := c.Query("apiKey")
		if url == "" || apiKey == "" {
			c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": "Missing url or apiKey"})
			return
		}
		err := testMediaConnection(url, apiKey, "sonarr")
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"success": false, "error": err.Error()})
		} else {
			c.JSON(http.StatusOK, gin.H{"success": true})
		}
	})
	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API endpoint for scheduled/queue status
	r.GET("/api/tasks/status", GetAllTasksStatus())
	r.POST("/api/tasks/force", TaskHandler())

	// Serve React static files and SPA fallback
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	// Serve static files for movie posters
	r.Static("/mediacover", MediaCoverPath)
	r.StaticFile("/logo.svg", "web/public/logo.svg")
	r.GET("/api/movies", getRadarrHandler)
	var defaultMoviePath string
	movieMappings, err := GetPathMappings("radarr")
	if err == nil && len(movieMappings) > 0 {
		for _, m := range movieMappings {
			if len(m) > 1 && m[1] != "" {
				defaultMoviePath = m[1]
				break
			}
		}
	}
	if defaultMoviePath == "" {
		defaultMoviePath = "/Movies"
	}
	r.GET("/api/movies/wanted", GetMediaWithoutTrailerExtraHandler("radarr", TrailarrRoot+"/movies_wanted.json", defaultMoviePath))
	r.GET("/api/series", getSonarrHandler)
	var defaultSeriesPath string
	seriesMappings, err := GetPathMappings("sonarr")
	if err == nil && len(seriesMappings) > 0 {
		for _, m := range seriesMappings {
			if len(m) > 1 && m[1] != "" {
				defaultSeriesPath = m[1]
				break
			}
		}
	}
	if defaultSeriesPath == "" {
		defaultSeriesPath = "/Series"
	}
	r.GET("/api/series/wanted", GetMediaWithoutTrailerExtraHandler("sonarr", TrailarrRoot+"/series_wanted.json", defaultSeriesPath))
	r.GET("/api/movies/:id/extras", getMovieExtrasHandler)
	r.GET("/api/series/:id/extras", getSeriesExtrasHandler)
	r.GET("/api/settings/radarr", GetSettingsHandler("radarr"))
	r.POST("/api/settings/radarr", SaveSettingsHandler("radarr"))
	r.GET("/api/settings/sonarr", GetSettingsHandler("sonarr"))
	r.POST("/api/settings/sonarr", SaveSettingsHandler("sonarr"))
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

	// Server-side file browser for directory picker
	r.GET("/api/files/list", ListServerFoldersHandler)
}
