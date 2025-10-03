package internal

import (
	"encoding/json"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Test TMDB API key endpoint
	r.GET("/api/test/tmdb", func(c *gin.Context) {
		apiKey := c.Query("apiKey")
		if apiKey == "" {
			respondError(c, http.StatusBadRequest, "Missing apiKey")
			return
		}
		testUrl := "https://api.themoviedb.org/3/configuration?api_key=" + apiKey
		resp, err := http.Get(testUrl)
		if CheckErrLog("Warn", "API", "TMDB testUrl http.Get failed", err) != nil {
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
			respondError(c, http.StatusBadRequest, "Missing url or apiKey")
			return
		}
		folders, err := FetchRootFolders(url, apiKey)
		if CheckErrLog("Warn", "API", "FetchRootFolders failed", err) != nil {
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
		if CheckErrLog("Warn", "API", "testMediaConnection "+provider+" failed", err) != nil {
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
	// Helper for default media path
	getDefaultPath := func(provider, fallback string) string {
		var mediaType MediaType
		switch provider {
		case "radarr":
			mediaType = MediaTypeMovie
		case "sonarr":
			mediaType = MediaTypeTV
		default:
			return fallback
		}
		mappings, err := GetPathMappings(mediaType)
		if CheckErrLog("Warn", "API", "GetPathMappings "+provider+" failed", err) == nil && len(mappings) > 0 {
			for _, m := range mappings {
				if len(m) > 1 && m[1] != "" {
					return m[1]
				}
			}
		}
		return fallback
	}
	// Group movies/series endpoints
	for _, media := range []struct {
		section      string
		cacheFile    string
		wantedFile   string
		fallbackPath string
		extrasType   MediaType
	}{
		{"movies", TrailarrRoot + "/movies.json", TrailarrRoot + "/movies_wanted.json", "/Movies", MediaTypeMovie},
		{"series", TrailarrRoot + "/series.json", TrailarrRoot + "/series_wanted.json", "/Series", MediaTypeTV},
	} {
		r.GET("/api/"+media.section, GetMediaHandler(media.section, media.cacheFile, "id"))
		var provider string
		if media.section == "movies" {
			provider = "radarr"
		} else {
			provider = "sonarr"
		}
		r.GET("/api/"+media.section+"/wanted", GetMediaWithoutTrailerExtraHandler(
			provider,
			media.wantedFile,
			getDefaultPath(provider, media.fallbackPath),
		))
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

	// Server-side file browser for directory picker
	r.GET("/api/files/list", ListServerFoldersHandler)
}
