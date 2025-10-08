package internal

func SyncSonarr() error {
	return SyncMedia(
		"sonarr",
		"/api/v3/series",
		SeriesJSONPath,
		func(m map[string]interface{}) bool {
			stats, ok := m["statistics"].(map[string]interface{})
			if !ok {
				return false
			}
			episodeFileCount, ok := stats["episodeFileCount"].(float64)
			return ok && episodeFileCount >= 1
		},
		MediaCoverPath+"/Series",
		[]string{"/poster-500.jpg", "/fanart-1280.jpg"},
	)
}
