package internal

const (
	TrailarrRoot             = "/var/lib/trailarr"
	ConfigPath               = TrailarrRoot + "/config/config.yml"
	MediaCoverPath           = TrailarrRoot + "/MediaCover"
	ErrInvalidSonarrSettings = "Invalid Sonarr settings"
	RemoteMediaCoverPath     = "/MediaCover/"
	HeaderApiKey             = "X-Api-Key"
	ErrInvalidRequest        = "invalid request"
	HeaderContentType        = "Content-Type"
)

type MediaType string

const (
	MediaTypeMovie MediaType = "movie"
	MediaTypeTV    MediaType = "tv"
)
