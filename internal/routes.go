package internal

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Serve static files for movie posters
	r.Static("/mediacover", "/var/lib/extrazarr/MediaCover")
	r.StaticFile("/logo.svg", "web/public/logo.svg")
	r.GET("/api/radarr/movies", getRadarrMoviesHandler)
	r.POST("/api/settings/radarr", saveRadarrSettingsHandler)
	r.GET("/api/settings/radarr", getRadarrSettingsHandler)
	r.GET("/api/extras/search", searchExtrasHandler)
	r.POST("/api/extras/download", downloadExtraHandler)
	r.GET("/api/extras/existing", existingExtrasHandler)
	r.GET("/api/sonarr/series", HandleSonarrSeries)
	r.POST("/api/settings/sonarr", saveSonarrSettingsHandler)
	r.GET("/api/settings/sonarr", getSonarrSettingsHandler)
	// Sonarr poster and banner proxy endpoints
	r.GET("/api/sonarr/poster/:seriesId", HandleSonarrPoster)
	r.GET("/api/sonarr/banner/:seriesId", HandleSonarrBanner)
	// Radarr poster and banner proxy endpoints
	r.GET("/api/radarr/poster/:movieId", HandleRadarrPoster)
	r.GET("/api/radarr/banner/:movieId", HandleRadarrBanner)
	// General settings (TMDB key)
	r.GET("/api/settings/general", getGeneralSettingsHandler)
	r.POST("/api/settings/general", saveGeneralSettingsHandler)
}
