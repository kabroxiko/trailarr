package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/kkdai/youtube/v2"
)

type ExtraDownloadMetadata struct {
	Title     string
	Type      string
	YouTubeID string
	FileName  string
	Status    string
	URL       string
}

func DownloadYouTubeExtra(moviePath, extraType, extraTitle, extraURL string) (*ExtraDownloadMetadata, error) {
	fmt.Printf("Downloading YouTube extra: type=%s, title=%s, url=%s\n", extraType, extraTitle, extraURL)
	// Extract YouTube ID
	youtubeID, err := ExtractYouTubeID(extraURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to extract YouTube ID: %w", err)
	}

	// Canonicalize the extra type for folder naming
	canonicalType := canonicalizeExtraType(extraType, extraTitle)
	outDir := filepath.Join(moviePath, canonicalType)
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return nil, fmt.Errorf("Failed to create output dir '%s': %w", outDir, err)
	}

	// Sanitize title for filename: only replace forbidden characters
	forbidden := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	safeTitle := extraTitle
	for _, c := range forbidden {
		safeTitle = strings.ReplaceAll(safeTitle, c, "_")
	}
	outFile := filepath.Join(outDir, fmt.Sprintf("%s.mp4", safeTitle))

	// Download using kkdai/youtube
	client := youtube.Client{}
	video, err := client.GetVideo(youtubeID)
	if err != nil {
		return nil, fmt.Errorf("Failed to get video info for YouTube ID '%s': %w", youtubeID, err)
	}
	formats := video.Formats.WithAudioChannels()
	if len(formats) == 0 {
		return nil, fmt.Errorf("No downloadable video format found for YouTube ID '%s'", youtubeID)
	}
	stream, _, err := client.GetStream(video, &formats[0])
	if err != nil {
		return nil, fmt.Errorf("Failed to get stream for YouTube ID '%s': %w", youtubeID, err)
	}
	f, err := os.Create(outFile)
	if err != nil {
		return nil, fmt.Errorf("Failed to create file '%s': %w", outFile, err)
	}
	defer f.Close()
	if _, err := io.Copy(f, stream); err != nil {
		return nil, fmt.Errorf("Failed to save video to '%s': %w", outFile, err)
	}

	meta := &ExtraDownloadMetadata{
		Title:     extraTitle,
		Type:      extraType,
		YouTubeID: youtubeID,
		FileName:  outFile,
		Status:    "downloaded",
		URL:       extraURL,
	}
	// Optionally, save metadata to a file (e.g., outFile+".json")
	metaFile := outFile + ".json"
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaFile, metaBytes, 0644)

	fmt.Printf("Downloaded %s to %s\n", extraTitle, outFile)
	return meta, nil
}
