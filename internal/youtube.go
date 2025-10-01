package internal

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	ytdlp "github.com/lrstanley/go-ytdlp"
)

type ExtraDownloadMetadata struct {
	MediaType  string // "movie" or "series"
	MediaTitle string // Movie or Series title
	ExtraType  string // e.g. "Trailer"
	ExtraTitle string // e.g. "Official Trailer"
	YouTubeID  string
	FileName   string
	Status     string
	URL        string
}

type RequestBody struct {
	Env   map[string]string `json:"env,omitempty"`
	Flags ytdlp.FlagConfig  `json:"flags"`
	Args  []string          `json:"args"`
}

func DownloadYouTubeExtra(mediaType, mediaTitle, extraType, extraTitle, extraURL string, forceDownload ...bool) (*ExtraDownloadMetadata, error) {
	force := false
	if len(forceDownload) > 0 {
		force = forceDownload[0]
	}

	if !force && !GetAutoDownloadExtras() {
		fmt.Printf("[DownloadYouTubeExtra] Auto download of extras is disabled by general settings. Skipping download for %s\n", extraTitle)
		return nil, nil
	}

	fmt.Printf("Downloading YouTube extra: mediaType=%s, mediaTitle=%s, type=%s, title=%s, url=%s\n", mediaType, mediaTitle, extraType, extraTitle, extraURL)

	// Extract YouTube ID and prepare paths
	youtubeID, err := ExtractYouTubeID(extraURL)
	if err != nil {
		return nil, fmt.Errorf("Failed to extract YouTube ID: %w", err)
	}

	downloadInfo, err := prepareDownloadInfo(mediaType, mediaTitle, extraType, extraTitle, youtubeID)
	if err != nil {
		return nil, err
	}

	// Check if extra is rejected or already exists
	if meta, err := checkExistingExtra(downloadInfo, extraURL); meta != nil || err != nil {
		return meta, err
	}

	// Perform the download
	return performDownload(downloadInfo, extraURL)
}

type downloadInfo struct {
	MediaType  string
	MediaTitle string
	OutDir     string
	OutFile    string
	TempDir    string
	TempFile   string
	YouTubeID  string
	ExtraType  string
	ExtraTitle string
	SafeTitle  string
}

func prepareDownloadInfo(mediaType, mediaTitle, extraType, extraTitle, youtubeID string) (*downloadInfo, error) {
	// Determine the base path based on media type
	var basePath string
	if mediaType == "movie" {
		basePath = filepath.Join("/home", "Movies", mediaTitle)
	} else if mediaType == "series" {
		basePath = filepath.Join("/home", "Series", mediaTitle)
	}

	canonicalType := canonicalizeExtraType(extraType, extraTitle)
	outDir := filepath.Join(basePath, canonicalType)

	// Sanitize title for filename
	forbidden := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	safeTitle := extraTitle
	for _, c := range forbidden {
		safeTitle = strings.ReplaceAll(safeTitle, c, "_")
	}

	outExt := "mkv"
	outFile := filepath.Join(outDir, fmt.Sprintf("%s.%s", safeTitle, outExt))

	// Create temp directory
	tempDir, err := os.MkdirTemp("", "yt-dlp-tmp-*")
	if err != nil {
		return nil, fmt.Errorf("Failed to create temp dir for yt-dlp: %w", err)
	}
	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s.%s", safeTitle, outExt))

	fmt.Printf("[DownloadYouTubeExtra] Resolved output directory: %s\n", outDir)
	fmt.Printf("[DownloadYouTubeExtra] Resolved safe title: %s\n", safeTitle)
	fmt.Printf("[DownloadYouTubeExtra][DEBUG] mediaType=%s, mediaTitle=%s, canonicalType=%s, outDir=%s, outFile=%s, tempDir=%s, tempFile=%s\n",
		mediaType, mediaTitle, canonicalType, outDir, outFile, tempDir, tempFile)

	return &downloadInfo{
		MediaType:  mediaType,
		MediaTitle: mediaTitle,
		OutDir:     outDir,
		OutFile:    outFile,
		TempDir:    tempDir,
		TempFile:   tempFile,
		YouTubeID:  youtubeID,
		ExtraType:  extraType,
		ExtraTitle: extraTitle,
		SafeTitle:  safeTitle,
	}, nil
}

func checkExistingExtra(info *downloadInfo, extraURL string) (*ExtraDownloadMetadata, error) {
	// Check if extra is in rejected_extras.json
	if meta := checkRejectedExtras(info, extraURL); meta != nil {
		return meta, nil
	}

	// Skip download if file already exists
	if _, err := os.Stat(info.OutFile); err == nil {
		fmt.Printf("[DownloadYouTubeExtra] File already exists, skipping: %s\n", info.OutFile)
		meta := &ExtraDownloadMetadata{
			ExtraTitle: info.ExtraTitle,
			ExtraType:  info.ExtraType,
			YouTubeID:  info.YouTubeID,
			FileName:   info.OutFile,
			Status:     "exists",
			URL:        extraURL,
		}
		return meta, nil
	}

	return nil, nil
}

func checkRejectedExtras(info *downloadInfo, extraURL string) *ExtraDownloadMetadata {
	rejectedPath := filepath.Join(TrailarrRoot, "rejected_extras.json")
	rejected := make([]map[string]string, 0)

	if data, err := os.ReadFile(rejectedPath); err == nil {
		_ = json.Unmarshal(data, &rejected)
		for _, r := range rejected {
			if r["url"] == extraURL {
				fmt.Printf("[DownloadYouTubeExtra] Extra is in rejected list, skipping: %s\n", info.ExtraTitle)
				return &ExtraDownloadMetadata{
					MediaType:  info.MediaType,
					MediaTitle: info.MediaTitle,
					ExtraType:  info.ExtraType,
					ExtraTitle: info.ExtraTitle,
					YouTubeID:  info.YouTubeID,
					FileName:   info.OutFile,
					Status:     "rejected",
					URL:        extraURL,
				}
			}
		}
		cleanupRejectedExtras(rejected, rejectedPath)
	}
	return nil
}

func cleanupRejectedExtras(rejected []map[string]string, rejectedPath string) {
	// Deduplicate rejected list by URL only
	unique := make([]map[string]string, 0)
	seen := make(map[string]bool)
	for _, r := range rejected {
		key := r["url"]
		if !seen[key] {
			unique = append(unique, r)
			seen[key] = true
		}
	}
	if len(unique) != len(rejected) {
		rejBytes, _ := json.MarshalIndent(unique, "", "  ")
		_ = os.WriteFile(rejectedPath, rejBytes, 0644)
	}
}

func performDownload(info *downloadInfo, extraURL string) (*ExtraDownloadMetadata, error) {
	defer os.RemoveAll(info.TempDir)

	// Setup yt-dlp
	ytdlp.MustInstall(context.Background(), nil)

	// First attempt with impersonation
	flags := createYtdlpFlags(info)
	cmd := ytdlp.New().
		SetWorkDir(info.TempDir).
		FormatSort("res,fps,codec,br").
		SetFlagConfig(&flags)

	output, err := cmd.Run(context.Background(), extraURL)
	if err != nil {
		// Check if error is related to impersonation
		if isImpersonationError(err) {
			fmt.Printf("[DownloadYouTubeExtra] Impersonation failed, retrying without impersonation: %s\n", extraURL)

			// Retry without impersonation
			flagsNoImpersonate := createYtdlpFlagsWithoutImpersonation(info)
			cmdRetry := ytdlp.New().
				SetWorkDir(info.TempDir).
				FormatSort("res,fps,codec,br").
				SetFlagConfig(&flagsNoImpersonate)

			output, err = cmdRetry.Run(context.Background(), extraURL)
		}
	}

	if err != nil {
		return nil, handleDownloadError(info, extraURL, err, output)
	}

	// Move file to final location
	if err := moveDownloadedFile(info); err != nil {
		return nil, err
	}

	// Create metadata
	return createSuccessMetadata(info, extraURL)
}

func shouldUseImpersonation() bool {
	// For now, always try to use impersonation
	// In the future, this could check for curl_cffi availability or user settings
	return true
}

func getImpersonationTarget() string {
	// Try different impersonation targets in order of preference
	// chrome is preferred as it's most commonly supported
	return "chrome"
}

func isImpersonationError(err error) bool {
	if err == nil {
		return false
	}
	errStr := err.Error()
	return strings.Contains(errStr, "Impersonate target") ||
		strings.Contains(errStr, "is not available") ||
		strings.Contains(errStr, "missing dependencies required to support this target")
}

func createYtdlpFlagsWithoutImpersonation(info *downloadInfo) ytdlp.FlagConfig {
	quiet := true
	noprogress := true
	writesubs := true
	writeautosubs := true
	embedsubs := true
	remuxvideo := "mkv"
	subformat := "srt"
	sublangs := "es.*"
	requestedformats := "best[height<=1080]"
	timeout := 3.0
	sleepInterval := 5.0
	maxDownloads := 5
	limitRate := "30M"
	sleepRequests := 3.0
	maxSleepInterval := 120.0

	cookiesPath := setupCookiesFile()

	// Create network flags WITHOUT impersonation
	networkFlags := ytdlp.FlagsNetwork{
		SocketTimeout: &timeout,
	}

	return ytdlp.FlagConfig{
		Network: networkFlags,
		VerbositySimulation: ytdlp.FlagsVerbositySimulation{
			Quiet:      &quiet,
			NoProgress: &noprogress,
		},
		Subtitle: ytdlp.FlagsSubtitle{
			WriteSubs:     &writesubs,
			WriteAutoSubs: &writeautosubs,
			SubFormat:     &subformat,
			SubLangs:      &sublangs,
		},
		PostProcessing: ytdlp.FlagsPostProcessing{
			EmbedSubs:  &embedsubs,
			RemuxVideo: &remuxvideo,
		},
		VideoFormat: ytdlp.FlagsVideoFormat{
			Format: &requestedformats,
		},
		Workarounds: ytdlp.FlagsWorkarounds{
			SleepInterval:    &sleepInterval,
			SleepRequests:    &sleepRequests,
			MaxSleepInterval: &maxSleepInterval,
		},
		VideoSelection: ytdlp.FlagsVideoSelection{
			MaxDownloads: &maxDownloads,
		},
		Filesystem: ytdlp.FlagsFilesystem{
			Output:  &info.TempFile,
			Cookies: cookiesPath,
		},
		Download: ytdlp.FlagsDownload{
			LimitRate: &limitRate,
		},
	}
}

func createYtdlpFlags(info *downloadInfo) ytdlp.FlagConfig {
	quiet := true
	noprogress := true
	writesubs := true
	writeautosubs := true
	embedsubs := true
	remuxvideo := "mkv"
	subformat := "srt"
	sublangs := "es.*"
	requestedformats := "best[height<=1080]"
	timeout := 3.0
	sleepInterval := 5.0
	maxDownloads := 5
	limitRate := "3M"
	sleepRequests := 3.0
	maxSleepInterval := 120.0

	cookiesPath := setupCookiesFile()

	// Create network flags with optional impersonation
	networkFlags := ytdlp.FlagsNetwork{
		SocketTimeout: &timeout,
	}

	// Try to enable impersonation if available
	if shouldUseImpersonation() {
		impersonate := getImpersonationTarget()
		networkFlags.Impersonate = &impersonate
	}

	return ytdlp.FlagConfig{
		Network: networkFlags,
		VerbositySimulation: ytdlp.FlagsVerbositySimulation{
			Quiet:      &quiet,
			NoProgress: &noprogress,
		},
		Subtitle: ytdlp.FlagsSubtitle{
			WriteSubs:     &writesubs,
			WriteAutoSubs: &writeautosubs,
			SubFormat:     &subformat,
			SubLangs:      &sublangs,
		},
		PostProcessing: ytdlp.FlagsPostProcessing{
			EmbedSubs:  &embedsubs,
			RemuxVideo: &remuxvideo,
		},
		VideoFormat: ytdlp.FlagsVideoFormat{
			Format: &requestedformats,
		},
		Workarounds: ytdlp.FlagsWorkarounds{
			SleepInterval:    &sleepInterval,
			SleepRequests:    &sleepRequests,
			MaxSleepInterval: &maxSleepInterval,
		},
		VideoSelection: ytdlp.FlagsVideoSelection{
			MaxDownloads: &maxDownloads,
		},
		Filesystem: ytdlp.FlagsFilesystem{
			Output:  &info.TempFile,
			Cookies: cookiesPath,
		},
		Download: ytdlp.FlagsDownload{
			LimitRate: &limitRate,
		},
	}
}

func setupCookiesFile() *string {
	fmt.Printf("[DownloadYouTubeExtra][DEBUG] TrailarrRoot: %s\n", TrailarrRoot)
	cookiesFile := filepath.Join(TrailarrRoot, "cookies.txt")

	if _, err := os.Stat(cookiesFile); err == nil {
		fmt.Printf("[DownloadYouTubeExtra] Using cookies file: %s\n", cookiesFile)
		return &cookiesFile
	}

	// Create an empty cookies.txt if it does not exist
	if f, createErr := os.Create(cookiesFile); createErr == nil {
		_ = f.Close()
		fmt.Printf("[DownloadYouTubeExtra] Created empty cookies.txt at %s\n", cookiesFile)
		return &cookiesFile
	} else {
		fmt.Printf("[DownloadYouTubeExtra] Could not create cookies.txt at %s: %v\n", cookiesFile, createErr)
		return nil
	}
}

func handleDownloadError(info *downloadInfo, extraURL string, err error, output *ytdlp.Result) error {
	errStr := err.Error()
	stderr := ""
	if output != nil {
		stderr = output.Stderr
	}

	reason := errStr
	if strings.Contains(errStr, "Did not get any data blocks") ||
		strings.Contains(errStr, "SABR") ||
		strings.Contains(stderr, "Did not get any data blocks") ||
		strings.Contains(stderr, "SABR") {
		reason = errStr
		if stderr != "" {
			reason += " | stderr: " + stderr
		}
	}

	addToRejectedExtras(info, extraURL, reason)
	return fmt.Errorf(reason+": %w", err)
}

func addToRejectedExtras(info *downloadInfo, extraURL, reason string) {
	rejectedPath := filepath.Join(TrailarrRoot, "rejected_extras.json")
	var rejectedList []map[string]string

	if data, err := os.ReadFile(rejectedPath); err == nil {
		_ = json.Unmarshal(data, &rejectedList)
	}

	// Check if already rejected
	for _, r := range rejectedList {
		if r["url"] == extraURL {
			return
		}
	}

	rejectedList = append(rejectedList, map[string]string{
		"media_type":  info.MediaType,
		"media_title": info.MediaTitle,
		"extra_type":  info.ExtraType,
		"extra_title": info.ExtraTitle,
		"url":         extraURL,
		"reason":      reason,
	})

	// Deduplicate
	unique := make([]map[string]string, 0)
	seen := make(map[string]bool)
	for _, r := range rejectedList {
		key := r["url"]
		if !seen[key] {
			unique = append(unique, r)
			seen[key] = true
		}
	}

	rejBytes, _ := json.MarshalIndent(unique, "", "  ")
	_ = os.WriteFile(rejectedPath, rejBytes, 0644)
}

func moveDownloadedFile(info *downloadInfo) error {
	if _, statErr := os.Stat(info.TempFile); statErr != nil {
		return fmt.Errorf("yt-dlp did not produce expected output file: %s", info.TempFile)
	}

	if err := os.MkdirAll(info.OutDir, 0755); err != nil {
		return fmt.Errorf("Failed to create output dir '%s': %w", info.OutDir, err)
	}

	if moveErr := os.Rename(info.TempFile, info.OutFile); moveErr != nil {
		return handleCrossDeviceMove(info.TempFile, info.OutFile, moveErr)
	}

	return nil
}

func handleCrossDeviceMove(tempFile, outFile string, moveErr error) error {
	if linkErr, ok := moveErr.(*os.LinkError); ok && strings.Contains(linkErr.Error(), "cross-device link") {
		return copyFileAcrossDevices(tempFile, outFile)
	}
	return fmt.Errorf("Failed to move downloaded file to output dir: %w", moveErr)
}

func copyFileAcrossDevices(tempFile, outFile string) error {
	in, err := os.Open(tempFile)
	if err != nil {
		return fmt.Errorf("Failed to open temp file for copy: %w", err)
	}
	defer in.Close()

	out, err := os.Create(outFile)
	if err != nil {
		return fmt.Errorf("Failed to create output file for copy: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return fmt.Errorf("Failed to copy file across devices: %w", err)
	}

	if err := out.Sync(); err != nil {
		return fmt.Errorf("Failed to sync output file: %w", err)
	}

	if rmErr := os.Remove(tempFile); rmErr != nil {
		fmt.Printf("[DownloadYouTubeExtra][WARN] Failed to remove temp file after copy: %v\n", rmErr)
	}

	return nil
}

func createSuccessMetadata(info *downloadInfo, extraURL string) (*ExtraDownloadMetadata, error) {
	meta := &ExtraDownloadMetadata{
		ExtraTitle: info.ExtraTitle,
		ExtraType:  info.ExtraType,
		YouTubeID:  info.YouTubeID,
		FileName:   info.OutFile,
		Status:     "downloaded",
		URL:        extraURL,
	}

	metaFile := info.OutFile + ".json"
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaFile, metaBytes, 0644)

	fmt.Printf("Downloaded %s to %s\n", info.ExtraTitle, info.OutFile)
	return meta, nil
}
