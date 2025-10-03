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

// Deduplicate a slice of maps by a given key
func DeduplicateByKey(list []map[string]string, key string) []map[string]string {
	seen := make(map[string]bool)
	unique := make([]map[string]string, 0, len(list))
	for _, item := range list {
		k := item[key]
		if !seen[k] {
			unique = append(unique, item)
			seen[k] = true
		}
	}
	return unique
}

type ExtraDownloadMetadata struct {
	MediaType  MediaType // "movie" or "series"
	MediaId    int       // Radarr or Sonarr ID as int
	MediaTitle string    // Movie or Series title
	ExtraType  string    // e.g. "Trailer"
	ExtraTitle string    // e.g. "Official Trailer"
	YouTubeID  string
	FileName   string
	Status     string
	URL        string
}

// NewExtraDownloadMetadata constructs an ExtraDownloadMetadata with status and all fields
func NewExtraDownloadMetadata(info *downloadInfo, extraURL string, status string) *ExtraDownloadMetadata {
	return &ExtraDownloadMetadata{
		MediaType:  info.MediaType,
		MediaId:    info.MediaId,
		MediaTitle: info.MediaTitle,
		ExtraTitle: info.ExtraTitle,
		ExtraType:  info.ExtraType,
		YouTubeID:  info.YouTubeID,
		FileName:   info.OutFile,
		Status:     status,
		URL:        extraURL,
	}
}

type RejectedExtra struct {
	MediaType  MediaType `json:"media_type"`
	MediaId    int       `json:"media_id"`
	MediaTitle string    `json:"media_title"`
	ExtraType  string    `json:"extra_type"`
	ExtraTitle string    `json:"extra_title"`
	URL        string    `json:"url"`
	Reason     string    `json:"reason"`
}

type RequestBody struct {
	Env   map[string]string `json:"env,omitempty"`
	Flags ytdlp.FlagConfig  `json:"flags"`
	Args  []string          `json:"args"`
}

func DownloadYouTubeExtra(mediaType MediaType, mediaId int, extraType, extraTitle, extraURL string, forceDownload ...bool) (*ExtraDownloadMetadata, error) {
	var downloadInfo *downloadInfo
	var err error

	force := false
	if len(forceDownload) > 0 {
		force = forceDownload[0]
	}

	if !force && !GetAutoDownloadExtras() {
		TrailarrLog("info", "YouTube", "Auto download of extras is disabled by general settings. Skipping download for %s", extraTitle)
		return nil, nil
	}

	// Lookup media title from cache for logging
	var mediaTitle string
	var cacheFile string
	cacheFile, _ = resolveCachePath(mediaType)
	if cacheFile != "" {
		items, _ := loadCache(cacheFile)
		for _, m := range items {
			idInt, ok := parseMediaID(m["id"])
			if ok && idInt == mediaId {
				if t, ok := m["title"].(string); ok {
					mediaTitle = t
					break
				}
			}
		}
	}
	TrailarrLog("info", "YouTube", "Downloading YouTube extra: mediaType=%s, mediaTitle=%s, type=%s, title=%s, url=%s", mediaType, mediaTitle, extraType, extraTitle, extraURL)

	// Extract YouTube ID and prepare paths
	youtubeID, err := ExtractYouTubeID(extraURL)
	if err != nil {
		return nil, fmt.Errorf("failed to extract YouTube ID: %w", err)
	}

	downloadInfo, err = prepareDownloadInfo(mediaType, mediaId, extraType, extraTitle, youtubeID)
	if err != nil {
		return nil, err
	}
	// Always clean up temp dir after download attempt
	defer func() {
		if downloadInfo != nil && downloadInfo.TempDir != "" {
			os.RemoveAll(downloadInfo.TempDir)
		}
	}()

	// Check if extra is rejected or already exists
	if meta, err := checkExistingExtra(downloadInfo, extraURL); meta != nil || err != nil {
		return meta, err
	}

	// Perform the download
	return performDownload(downloadInfo, extraURL)
}

type downloadInfo struct {
	MediaType  MediaType
	MediaId    int
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

func prepareDownloadInfo(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeID string) (*downloadInfo, error) {
	// Robust base path resolution using cache and path mappings
	var basePath string
	var mappings [][]string
	var err error
	var cacheFile string
	var mediaTitle string

	// Step 1: Resolve cache path
	cacheFile, _ = resolveCachePath(mediaType)

	// Lookup media title from cache (mimic lookupMediaTitle from extras.go)
	if cacheFile != "" {
		items, _ := loadCache(cacheFile)
		for _, m := range items {
			idInt, ok := parseMediaID(m["id"])
			if ok && idInt == mediaId {
				if t, ok := m["title"].(string); ok {
					mediaTitle = t
					break
				}
			}
		}
	}

	// Step 2: Get path mappings using GetPathMappings
	mappings, err = GetPathMappings(mediaType)
	if err != nil {
		TrailarrLog("error", "YouTube", "Failed to get path mappings: %v", err)
		mappings = [][]string{}
	}

	var mappedMediaPath string
	if err == nil && cacheFile != "" {
		// Step 3: Look up media path from cache using mediaId
		mediaPath, lookupErr := FindMediaPathByID(cacheFile, mediaId)
		if lookupErr == nil && mediaPath != "" && len(mappings) > 0 {
			// Step 4: Apply path mappings to convert root folder path
			mappedMediaPath = mediaPath
			for _, m := range mappings {
				if len(m) > 1 && strings.HasPrefix(mediaPath, m[0]) {
					mappedMediaPath = m[1] + mediaPath[len(m[0]):]
					break
				}
			}
		}
	}

	if mappedMediaPath != "" {
		basePath = mappedMediaPath
	} else if len(mappings) > 0 && len(mappings[0]) > 1 && mappings[0][1] != "" {
		basePath = mediaTitle
		if mediaTitle != "" {
			basePath = filepath.Join(mappings[0][1], mediaTitle)
		} else {
			basePath = mappings[0][1]
		}
	} else {
		if mediaTitle != "" {
			basePath = mediaTitle
		} else {
			basePath = ""
		}
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
		TrailarrLog("error", "YouTube", "Failed to create temp dir for yt-dlp: %v", err)
		return nil, fmt.Errorf("failed to create temp dir for yt-dlp: %w", err)
	}
	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s.%s", safeTitle, outExt))

	TrailarrLog("debug", "YouTube", "Resolved output directory: %s", outDir)
	TrailarrLog("debug", "YouTube", "Resolved safe title: %s", safeTitle)
	TrailarrLog("debug", "YouTube", "mediaType=%s, mediaTitle=%s, canonicalType=%s, outDir=%s, outFile=%s, tempDir=%s, tempFile=%s",
		mediaType, mediaTitle, canonicalType, outDir, outFile, tempDir, tempFile)

	return &downloadInfo{
		MediaType:  mediaType,
		MediaId:    mediaId,
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
		TrailarrLog("info", "YouTube", "File already exists, skipping: %s", info.OutFile)
		return NewExtraDownloadMetadata(info, extraURL, "exists"), nil
	}

	return nil, nil
}

func checkRejectedExtras(info *downloadInfo, extraURL string) *ExtraDownloadMetadata {
	rejectedPath := filepath.Join(TrailarrRoot, "rejected_extras.json")
	rejected := make([]map[string]string, 0)

	if err := ReadJSONFile(rejectedPath, &rejected); err == nil {
		for _, r := range rejected {
			if r["url"] == extraURL {
				TrailarrLog("info", "YouTube", "Extra is in rejected list, skipping: %s", info.ExtraTitle)
				return NewExtraDownloadMetadata(info, extraURL, "rejected")
			}
		}
		cleanupRejectedExtras(rejected, rejectedPath)
	}
	return nil
}

func cleanupRejectedExtras(rejected []map[string]string, rejectedPath string) {
	// Deduplicate by URL
	unique := DeduplicateByKey(rejected, "url")
	if len(unique) != len(rejected) {
		_ = WriteJSONFile(rejectedPath, unique)
	}
}

func performDownload(info *downloadInfo, extraURL string) (*ExtraDownloadMetadata, error) {

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
	cfg, _ := GetYtdlpFlagsConfig()
	cookiesPath := setupCookiesFile()
	networkFlags := ytdlp.FlagsNetwork{
		SocketTimeout: &cfg.Timeout,
	}
	return ytdlp.FlagConfig{
		Network: networkFlags,
		VerbositySimulation: ytdlp.FlagsVerbositySimulation{
			Quiet:      &cfg.Quiet,
			NoProgress: &cfg.NoProgress,
		},
		Subtitle: ytdlp.FlagsSubtitle{
			WriteSubs:     &cfg.WriteSubs,
			WriteAutoSubs: &cfg.WriteAutoSubs,
			SubFormat:     &cfg.SubFormat,
			SubLangs:      &cfg.SubLangs,
		},
		PostProcessing: ytdlp.FlagsPostProcessing{
			EmbedSubs:  &cfg.EmbedSubs,
			RemuxVideo: &cfg.RemuxVideo,
		},
		VideoFormat: ytdlp.FlagsVideoFormat{
			Format: &cfg.RequestedFormats,
		},
		Workarounds: ytdlp.FlagsWorkarounds{
			SleepInterval:    &cfg.SleepInterval,
			SleepRequests:    &cfg.SleepRequests,
			MaxSleepInterval: &cfg.MaxSleepInterval,
		},
		VideoSelection: ytdlp.FlagsVideoSelection{
			MaxDownloads: &cfg.MaxDownloads,
		},
		Filesystem: ytdlp.FlagsFilesystem{
			Output:  &info.TempFile,
			Cookies: cookiesPath,
		},
		Download: ytdlp.FlagsDownload{
			LimitRate: &cfg.LimitRate,
		},
	}
}

func createYtdlpFlags(info *downloadInfo) ytdlp.FlagConfig {
	cfg, _ := GetYtdlpFlagsConfig()
	cookiesPath := setupCookiesFile()
	networkFlags := ytdlp.FlagsNetwork{
		SocketTimeout: &cfg.Timeout,
	}
	// Try to enable impersonation if available
	if shouldUseImpersonation() {
		impersonate := getImpersonationTarget()
		networkFlags.Impersonate = &impersonate
	}
	return ytdlp.FlagConfig{
		Network: networkFlags,
		VerbositySimulation: ytdlp.FlagsVerbositySimulation{
			Quiet:      &cfg.Quiet,
			NoProgress: &cfg.NoProgress,
		},
		Subtitle: ytdlp.FlagsSubtitle{
			WriteSubs:     &cfg.WriteSubs,
			WriteAutoSubs: &cfg.WriteAutoSubs,
			SubFormat:     &cfg.SubFormat,
			SubLangs:      &cfg.SubLangs,
		},
		PostProcessing: ytdlp.FlagsPostProcessing{
			EmbedSubs:  &cfg.EmbedSubs,
			RemuxVideo: &cfg.RemuxVideo,
		},
		VideoFormat: ytdlp.FlagsVideoFormat{
			Format: &cfg.RequestedFormats,
		},
		Workarounds: ytdlp.FlagsWorkarounds{
			SleepInterval:    &cfg.SleepInterval,
			SleepRequests:    &cfg.SleepRequests,
			MaxSleepInterval: &cfg.MaxSleepInterval,
		},
		VideoSelection: ytdlp.FlagsVideoSelection{
			MaxDownloads: &cfg.MaxDownloads,
		},
		Filesystem: ytdlp.FlagsFilesystem{
			Output:  &info.TempFile,
			Cookies: cookiesPath,
		},
		Download: ytdlp.FlagsDownload{
			LimitRate: &cfg.LimitRate,
		},
	}
}

func setupCookiesFile() *string {
	TrailarrLog("debug", "YouTube", "TrailarrRoot: %s", TrailarrRoot)
	cookiesFile := filepath.Join(TrailarrRoot, "cookies.txt")

	if _, err := os.Stat(cookiesFile); err == nil {
		TrailarrLog("info", "YouTube", "Using cookies file: %s", cookiesFile)
		return &cookiesFile
	}

	// Create an empty cookies.txt if it does not exist
	if f, createErr := os.Create(cookiesFile); createErr == nil {
		_ = f.Close()
		TrailarrLog("info", "YouTube", "Created empty cookies.txt at %s", cookiesFile)
		return &cookiesFile
	} else {
		TrailarrLog("error", "YouTube", "Could not create cookies.txt at %s: %v", cookiesFile, createErr)
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

	TrailarrLog("error", "YouTube", "Download failed for %s: %s", extraURL, reason)
	addToRejectedExtras(info, extraURL, reason)
	return fmt.Errorf(reason+": %w", err)
}

func addToRejectedExtras(info *downloadInfo, extraURL, reason string) {
	rejectedPath := filepath.Join(TrailarrRoot, "rejected_extras.json")
	var rejectedList []RejectedExtra

	_ = ReadJSONFile(rejectedPath, &rejectedList)

	// Check if already rejected
	for _, r := range rejectedList {
		if r.URL == extraURL {
			return
		}
	}

	rejectedList = append(rejectedList, RejectedExtra{
		MediaType:  info.MediaType,
		MediaId:    info.MediaId,
		MediaTitle: info.MediaTitle,
		ExtraType:  info.ExtraType,
		ExtraTitle: info.ExtraTitle,
		URL:        extraURL,
		Reason:     reason,
	})

	// Deduplicate by URL
	tempList := make([]map[string]string, 0, len(rejectedList))
	for _, r := range rejectedList {
		tempList = append(tempList, map[string]string{"url": r.URL})
	}
	uniqueURLs := DeduplicateByKey(tempList, "url")
	finalList := make([]RejectedExtra, 0, len(uniqueURLs))
	for _, u := range uniqueURLs {
		for _, r := range rejectedList {
			if r.URL == u["url"] {
				finalList = append(finalList, r)
				break
			}
		}
	}
	_ = WriteJSONFile(rejectedPath, finalList)
}

func moveDownloadedFile(info *downloadInfo) error {
	if _, statErr := os.Stat(info.TempFile); statErr != nil {
		TrailarrLog("error", "YouTube", "yt-dlp did not produce expected output file: %s", info.TempFile)
		return fmt.Errorf("yt-dlp did not produce expected output file: %s", info.TempFile)
	}

	if err := os.MkdirAll(info.OutDir, 0755); err != nil {
		TrailarrLog("error", "YouTube", "Failed to create output dir '%s': %v", info.OutDir, err)
		return fmt.Errorf("failed to create output dir '%s': %w", info.OutDir, err)
	}

	if moveErr := os.Rename(info.TempFile, info.OutFile); moveErr != nil {
		TrailarrLog("warn", "YouTube", "Rename failed, attempting cross-device move: %v", moveErr)
		return handleCrossDeviceMove(info.TempFile, info.OutFile, moveErr)
	}

	return nil
}

func handleCrossDeviceMove(tempFile, outFile string, moveErr error) error {
	if linkErr, ok := moveErr.(*os.LinkError); ok && strings.Contains(linkErr.Error(), "cross-device link") {
		TrailarrLog("warn", "YouTube", "Cross-device link error, copying file instead: %v", moveErr)
		return copyFileAcrossDevices(tempFile, outFile)
	}
	TrailarrLog("error", "YouTube", "Failed to move downloaded file to output dir: %v", moveErr)
	return fmt.Errorf("failed to move downloaded file to output dir: %w", moveErr)
}

func copyFileAcrossDevices(tempFile, outFile string) error {
	in, err := os.Open(tempFile)
	if err != nil {
		TrailarrLog("error", "YouTube", "Failed to open temp file for copy: %v", err)
		return fmt.Errorf("failed to open temp file for copy: %w", err)
	}
	defer in.Close()

	out, err := os.Create(outFile)
	if err != nil {
		TrailarrLog("error", "YouTube", "Failed to create output file for copy: %v", err)
		return fmt.Errorf("failed to create output file for copy: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		TrailarrLog("error", "YouTube", "Failed to copy file across devices: %v", err)
		return fmt.Errorf("failed to copy file across devices: %w", err)
	}

	if err := out.Sync(); err != nil {
		TrailarrLog("error", "YouTube", "Failed to sync output file: %v", err)
		return fmt.Errorf("failed to sync output file: %w", err)
	}

	if rmErr := os.Remove(tempFile); rmErr != nil {
		TrailarrLog("warn", "YouTube", "Failed to remove temp file after copy: %v", rmErr)
	}

	return nil
}

func createSuccessMetadata(info *downloadInfo, extraURL string) (*ExtraDownloadMetadata, error) {
	meta := NewExtraDownloadMetadata(info, extraURL, "downloaded")

	metaFile := info.OutFile + ".json"
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaFile, metaBytes, 0644)

	TrailarrLog("info", "YouTube", "Downloaded %s to %s", info.ExtraTitle, info.OutFile)
	return meta, nil
}
