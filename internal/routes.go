package internal

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Serve static files for movie posters
	r.Static("/mediacover", "/var/lib/extrazarr/MediaCover")
	r.StaticFile("/logo.svg", "web/public/logo.svg")
	r.GET("/api/radarr/movies", getRadarrMoviesHandler)
	r.GET("/api/sonarr/series", getSonarrSeriesHandler)
	r.GET("/api/settings/radarr", getRadarrSettingsHandler)
	r.POST("/api/settings/radarr", saveRadarrSettingsHandler)
	r.GET("/api/settings/sonarr", getSonarrSettingsHandler)
	r.POST("/api/settings/sonarr", saveSonarrSettingsHandler)
	r.GET("/api/extras/search", searchExtrasHandler)
	r.POST("/api/extras/download", downloadExtraHandler)
	r.GET("/api/extras/existing", existingExtrasHandler)
	// Sonarr poster and banner proxy endpoints
	r.GET("/api/sonarr/poster/:serieId", getSonarrPosterHandler)
	r.GET("/api/sonarr/banner/:serieId", getSonarrBannerHandler)
	// Radarr poster and banner proxy endpoints
	r.GET("/api/radarr/poster/:movieId", getRadarrPosterHandler)
	r.GET("/api/radarr/banner/:movieId", getRadarrBannerHandler)
	// General settings (TMDB key)
	r.GET("/api/settings/general", getGeneralSettingsHandler)
	r.POST("/api/settings/general", saveGeneralSettingsHandler)
}
