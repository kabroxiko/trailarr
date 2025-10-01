package internal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"gopkg.in/yaml.v3"
)

func SyncSonarrSeriesAndMediaCover() error {
	var syncErr error
	data, err := os.ReadFile(ConfigPath)
	if err != nil {
		fmt.Println("[SyncSonarr] Sonarr settings not found")
		return fmt.Errorf("Sonarr settings not found: %w", err)
	}
	var allSettings struct {
		Sonarr struct {
			URL    string `yaml:"url"`
			APIKey string `yaml:"apiKey"`
		} `yaml:"sonarr"`
	}
	if err := yaml.Unmarshal(data, &allSettings); err != nil {
		fmt.Println("[SyncSonarr] Invalid Sonarr settings")
		return fmt.Errorf("Invalid Sonarr settings: %w", err)
	}
	settings := allSettings.Sonarr
	cachePath := SeriesCachePath
	req, err := http.NewRequest("GET", settings.URL+"/api/v3/series", nil)
	if err != nil {
		fmt.Println("[SyncSonarr] Error creating request:", err)
		return fmt.Errorf("Error creating request: %w", err)
	}
	req.Header.Set(HeaderApiKey, settings.APIKey)
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("[SyncSonarr] Error fetching series:", err)
		return fmt.Errorf("Error fetching series: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		fmt.Println("[SyncSonarr] Sonarr API error:", resp.StatusCode)
		return fmt.Errorf("Sonarr API error: %d", resp.StatusCode)
	}
	var allSeries []map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&allSeries); err != nil {
		fmt.Println("[SyncSonarr] Failed to decode Sonarr response:", err)
		return fmt.Errorf("Failed to decode Sonarr response: %w", err)
	}
	// Filter only series with downloaded episodes
	series := make([]map[string]interface{}, 0)
	for _, s := range allSeries {
		stats, ok := s["statistics"].(map[string]interface{})
		if !ok {
			continue
		}
		episodeFileCount, ok := stats["episodeFileCount"].(float64)
		if !ok || episodeFileCount < 1 {
			continue
		}
		series = append(series, s)
	}
	cacheData, _ := json.MarshalIndent(series, "", "  ")
	_ = os.WriteFile(cachePath, cacheData, 0644)
	fmt.Println("[SyncSonarr] Synced", len(series), "downloaded series to cache.")
	return syncErr
}
