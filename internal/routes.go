package internal

import (
	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.Engine) {
	// Serve static files for movie posters
	r.Static("/mediacover", TrailarrRoot+"/MediaCover")
	r.StaticFile("/logo.svg", "web/public/logo.svg")
	r.GET("/api/movies", getRadarrHandler)
	var defaultMoviePath string
	movieMappings, err := GetPathMappings("radarr")
	if err == nil && len(movieMappings) > 0 {
		for _, m := range movieMappings {
			if m[1] != "" {
				defaultMoviePath = m[1]
				break
			}
		}
	}
	if defaultMoviePath == "" {
		defaultMoviePath = "/mnt/unionfs/Media/Movies"
	}
	r.GET("/api/movies/no_trailer_extra", GetMediaWithoutTrailerExtraHandler("radarr", MoviesCachePath, defaultMoviePath))
	r.GET("/api/series", getSonarrHandler)
	var defaultSeriesPath string
	seriesMappings, err := GetPathMappings("sonarr")
	if err == nil && len(seriesMappings) > 0 {
		for _, m := range seriesMappings {
			if m[1] != "" {
				defaultSeriesPath = m[1]
				break
			}
		}
	}
	if defaultSeriesPath == "" {
		defaultSeriesPath = "/mnt/unionfs/Media/TV"
	}
	r.GET("/api/series/no_trailer_extra", GetMediaWithoutTrailerExtraHandler("sonarr", SeriesCachePath, defaultSeriesPath))
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

	// Server-side file browser for directory picker
	r.GET("/api/files/list", ListServerFoldersHandler)
}
