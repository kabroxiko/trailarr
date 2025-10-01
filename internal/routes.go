package internal

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Health check
	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// API endpoint for scheduled/queue status
	r.GET("/api/tasks/status", GetAllTasksStatus())
	r.POST("/api/tasks/force", ForceTaskHandler())

	// Serve React static files and SPA fallback
	r.Static("/assets", "./web/dist/assets")
	r.StaticFile("/", "./web/dist/index.html")
	r.NoRoute(func(c *gin.Context) {
		c.File("./web/dist/index.html")
	})

	// Serve static files for movie posters
	r.Static("/mediacover", TrailarrRoot+"/MediaCover")
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
