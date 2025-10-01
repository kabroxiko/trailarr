package internal

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Serve static files for movie posters
	r.Static("/mediacover", "/var/lib/extrazarr/MediaCover")
	r.StaticFile("/logo.svg", "web/public/logo.svg")
	r.GET("/api/movies", getRadarrHandler)
	r.GET("/api/series", getSonarrHandler)
	r.GET("/api/movies/:id/extras", getMovieExtrasHandler)
	r.GET("/api/series/:id/extras", getSeriesExtrasHandler)
	r.GET("/api/settings/radarr", getRadarrSettingsHandler)
	r.POST("/api/settings/radarr", saveRadarrSettingsHandler)
	r.GET("/api/settings/sonarr", getSonarrSettingsHandler)
	r.POST("/api/settings/sonarr", saveSonarrSettingsHandler)
	r.POST("/api/extras/download", downloadExtraHandler)
	r.DELETE("/api/extras", deleteExtraHandler)
	r.GET("/api/extras/existing", existingExtrasHandler)
	r.GET("/api/history", historyHandler)
	// Posters and banners are now served directly from /mediacover static path
	// General settings (TMDB key)
	r.GET("/api/settings/general", getGeneralSettingsHandler)
	r.POST("/api/settings/general", saveGeneralSettingsHandler)
}
