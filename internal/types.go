package internal

// Common TMDB extra types (singular)
var TMDBExtraTypes = []string{
	"Trailer",
	"Teaser",
	"Clip",
	"Featurette",
	"Behind the Scene",
	"Bloopers",
	"Opening Credit",
	"Recap",
	"Interview",
	"Scene",
	"Promo",
	"Short",
	"Music Video",
	"Commercial",
	"Other",
}

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
