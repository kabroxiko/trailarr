package internal

func SyncRadarr() error {
	removeRejectedExtrasWithReasons()
	return SyncMedia(
		"radarr",
		"/api/v3/movie",
		MoviesJSONPath,
		func(m map[string]interface{}) bool {
			hasFile, ok := m["hasFile"].(bool)
			return ok && hasFile
		},
		MediaCoverPath+"/Movies",
		[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
	)
}
