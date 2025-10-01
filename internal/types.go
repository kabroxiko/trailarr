package internal

const (
	TrailarrRoot             = "/var/lib/trailarr"
	ConfigPath               = TrailarrRoot + "/config/config.yml"
	MoviesCachePath          = TrailarrRoot + "/movies_cache.json"
	MediaCoverPath           = TrailarrRoot + "/MediaCover/"
	SeriesCachePath          = TrailarrRoot + "/series_cache.json"
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
