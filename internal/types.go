package internal

const (
	ConfigPath               = "/var/lib/extrazarr/config/config.yml"
	MoviesCachePath          = "/var/lib/extrazarr/movies_cache.json"
	MediaCoverPath           = "/var/lib/extrazarr/MediaCover/"
	SeriesCachePath          = "/var/lib/extrazarr/series_cache.json"
	ErrInvalidSonarrSettings = "Invalid Sonarr settings"
	RemoteMediaCoverPath     = "/MediaCover/"
	HeaderApiKey             = "X-Api-Key"
	ErrInvalidRequest        = "invalid request"
	HeaderContentType        = "Content-Type"
)

type SonarrSeries struct {
	ID       int    `json:"id"`
	Title    string `json:"title"`
	Year     int    `json:"year"`
	Path     string `json:"path"`
	Overview string `json:"overview"`
}

// HistoryEvent struct is now in history.go
