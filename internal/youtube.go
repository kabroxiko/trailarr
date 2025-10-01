package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	go_ytdlp "github.com/lrstanley/go-ytdlp"
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
	if !GetAutoDownloadExtras() {
		fmt.Printf("[DownloadYouTubeExtra] Auto download of extras is disabled by general settings. Skipping download for %s\n", extraTitle)
		return nil, nil
	}
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

	// Download using yt-dlp via go-ytdlp
	// Correct usage based on go-ytdlp docs
	// https://github.com/lrstanley/go-ytdlp#simple
	// MustInstall ensures yt-dlp binary is available
	go_ytdlp.MustInstall(context.Background(), nil)
	cmd := go_ytdlp.New().
		Output(outFile).
		FormatSort("res,ext:mp4:m4a").
		RecodeVideo("mp4")

	_, err = cmd.Run(context.Background(), extraURL)
	if err != nil {
		return nil, fmt.Errorf("yt-dlp download failed: %w", err)
	}

	meta := &ExtraDownloadMetadata{
		Title:     extraTitle,
		Type:      extraType,
		YouTubeID: youtubeID,
		FileName:  outFile,
		Status:    "downloaded",
		URL:       extraURL,
	}
	metaFile := outFile + ".json"
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaFile, metaBytes, 0644)

	fmt.Printf("Downloaded %s to %s\n", extraTitle, outFile)
	return meta, nil
}
