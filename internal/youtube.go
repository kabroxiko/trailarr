package internal

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type YtdlpFlagsConfig struct {
	Quiet              bool    `yaml:"quiet" json:"quiet"`
	NoProgress         bool    `yaml:"noprogress" json:"noprogress"`
	WriteSubs          bool    `yaml:"writesubs" json:"writesubs"`
	WriteAutoSubs      bool    `yaml:"writeautosubs" json:"writeautosubs"`
	EmbedSubs          bool    `yaml:"embedsubs" json:"embedsubs"`
	RemuxVideo         string  `yaml:"remuxvideo" json:"remuxvideo"`
	SubFormat          string  `yaml:"subformat" json:"subformat"`
	SubLangs           string  `yaml:"sublangs" json:"sublangs"`
	RequestedFormats   string  `yaml:"requestedformats" json:"requestedformats"`
	Timeout            float64 `yaml:"timeout" json:"timeout"`
	SleepInterval      float64 `yaml:"sleepInterval" json:"sleepInterval"`
	MaxDownloads       int     `yaml:"maxDownloads" json:"maxDownloads"`
	LimitRate          string  `yaml:"limitRate" json:"limitRate"`
	SleepRequests      float64 `yaml:"sleepRequests" json:"sleepRequests"`
	MaxSleepInterval   float64 `yaml:"maxSleepInterval" json:"maxSleepInterval"`
	CookiesFromBrowser string  `yaml:"cookiesFromBrowser" json:"cookiesFromBrowser"`
}

func DefaultYtdlpFlagsConfig() YtdlpFlagsConfig {
	return YtdlpFlagsConfig{
		Quiet:              false,
		NoProgress:         false,
		WriteSubs:          true,
		WriteAutoSubs:      true,
		EmbedSubs:          true,
		RemuxVideo:         "mkv",
		SubFormat:          "srt",
		SubLangs:           "es.*",
		RequestedFormats:   "best[height<=1080]",
		Timeout:            3.0,
		SleepInterval:      5.0,
		MaxDownloads:       5,
		LimitRate:          "30M",
		SleepRequests:      3.0,
		MaxSleepInterval:   120.0,
		CookiesFromBrowser: "chrome",
	}
}

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
func NewExtraDownloadMetadata(info *downloadInfo, youtubeId string, status string) *ExtraDownloadMetadata {
	return &ExtraDownloadMetadata{
		MediaType:  info.MediaType,
		MediaId:    info.MediaId,
		MediaTitle: info.MediaTitle,
		ExtraTitle: info.ExtraTitle,
		ExtraType:  info.ExtraType,
		YouTubeID:  info.YouTubeID,
		FileName:   info.OutFile,
		Status:     status,
		URL:        youtubeId,
	}
}

type RejectedExtra struct {
	MediaType  MediaType `json:"media_type"`
	MediaId    int       `json:"media_id"`
	MediaTitle string    `json:"media_title"`
	ExtraType  string    `json:"extra_type"`
	ExtraTitle string    `json:"extra_title"`
	YoutubeId  string    `json:"youtube_id"`
	Reason     string    `json:"reason"`
}

// No longer needed

func DownloadYouTubeExtra(mediaType MediaType, mediaId int, extraType, extraTitle, youtubeId string, forceDownload ...bool) (*ExtraDownloadMetadata, error) {
	TrailarrLog(DEBUG, "YouTube", "DownloadYouTubeExtra called with mediaType=%s, mediaId=%d, extraType=%s, extraTitle=%s, youtubeId=%s, forceDownload=%v",
		mediaType, mediaId, extraType, extraTitle, youtubeId, forceDownload)
	var downloadInfo *downloadInfo
	var err error

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
	TrailarrLog(INFO, "YouTube", "Downloading YouTube extra: mediaType=%s, mediaTitle=%s, type=%s, title=%s, youtubeId=%s",
		mediaType, mediaTitle, extraType, extraTitle, youtubeId)

	downloadInfo, err = prepareDownloadInfo(mediaType, mediaId, extraType, extraTitle, youtubeId)
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
	if meta, err := checkExistingExtra(downloadInfo, youtubeId); meta != nil || err != nil {
		return meta, err
	}

	// Perform the download
	return performDownload(downloadInfo, youtubeId)
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
		TrailarrLog(ERROR, "YouTube", "Failed to get path mappings: %v", err)
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
		TrailarrLog(ERROR, "YouTube", "Failed to create temp dir for yt-dlp: %v", err)
		return nil, fmt.Errorf("failed to create temp dir for yt-dlp: %w", err)
	}
	tempFile := filepath.Join(tempDir, fmt.Sprintf("%s.%s", safeTitle, outExt))

	TrailarrLog(DEBUG, "YouTube", "Resolved output directory: %s", outDir)
	TrailarrLog(DEBUG, "YouTube", "Resolved safe title: %s", safeTitle)
	TrailarrLog(DEBUG, "YouTube", "mediaType=%s, mediaTitle=%s, canonicalType=%s, outDir=%s, outFile=%s, tempDir=%s, tempFile=%s",
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

func checkExistingExtra(info *downloadInfo, youtubeId string) (*ExtraDownloadMetadata, error) {
	// Check if extra is in rejected_extras.json
	if meta := checkRejectedExtras(info, youtubeId); meta != nil {
		return meta, nil
	}

	// Skip download if file already exists
	if _, err := os.Stat(info.OutFile); err == nil {
		TrailarrLog(INFO, "YouTube", "File already exists, skipping: %s", info.OutFile)
		return NewExtraDownloadMetadata(info, youtubeId, "exists"), nil
	}

	return nil, nil
}

func checkRejectedExtras(info *downloadInfo, youtubeId string) *ExtraDownloadMetadata {
	rejectedPath := RejectedExtrasPath
	rejected := make([]map[string]string, 0)

	if err := ReadJSONFile(rejectedPath, &rejected); err == nil {
		for _, r := range rejected {
			if r["url"] == youtubeId {
				TrailarrLog(INFO, "YouTube", "Extra is in rejected list, skipping: %s", info.ExtraTitle)
				return NewExtraDownloadMetadata(info, youtubeId, "rejected")
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

func performDownload(info *downloadInfo, youtubeId string) (*ExtraDownloadMetadata, error) {

	// Build yt-dlp command args
	args := buildYtDlpArgs(info, youtubeId, true)

	// Debug: print the full yt-dlp command line
	fullCmd := "yt-dlp " + strings.Join(args, " ")
	TrailarrLog(INFO, "YouTube", "yt-dlp command: %s", fullCmd)

	cmd := exec.Command("yt-dlp", args...)
	cmd.Dir = info.TempDir
	output, err := cmd.CombinedOutput()

	if err != nil && isImpersonationErrorNative(string(output)) {
		fmt.Printf("[DownloadYouTubeExtra] Impersonation failed, retrying without impersonation: %s\n", youtubeId)
		args = buildYtDlpArgs(info, youtubeId, false)
		cmd = exec.Command("yt-dlp", args...)
		cmd.Dir = info.TempDir
		output, err = cmd.CombinedOutput()
	}

	if len(output) > 0 {
		for _, line := range strings.Split(string(output), "\n") {
			if strings.TrimSpace(line) != "" {
				TrailarrLog(DEBUG, "YouTube", "yt-dlp output for %s: %s", youtubeId, line)
			}
		}
	}
	if err != nil {
		return nil, handleDownloadErrorNative(info, youtubeId, err, string(output))
	}

	// Move file to final location
	if err := moveDownloadedFile(info); err != nil {
		return nil, err
	}

	// Create metadata
	return createSuccessMetadata(info, youtubeId)
}

func isImpersonationErrorNative(output string) bool {
	return strings.Contains(output, "Impersonate target") ||
		strings.Contains(output, "is not available") ||
		strings.Contains(output, "missing dependencies required to support this target")
}

func buildYtDlpArgs(info *downloadInfo, youtubeId string, impersonate bool) []string {
	cfg, _ := GetYtdlpFlagsConfig()
	args := []string{
		"--no-progress",
		"--quiet",
		"--write-subs",
		"--write-auto-subs",
		"--embed-subs",
		"--sub-format", cfg.SubFormat,
		"--sub-langs", cfg.SubLangs,
		"--remux-video", cfg.RemuxVideo,
		"--format", cfg.RequestedFormats,
		"--output", info.TempFile,
		"--max-downloads", fmt.Sprintf("%d", cfg.MaxDownloads),
		"--limit-rate", cfg.LimitRate,
		"--sleep-interval", fmt.Sprintf("%.0f", cfg.SleepInterval),
		"--sleep-requests", fmt.Sprintf("%.0f", cfg.SleepRequests),
		"--max-sleep-interval", fmt.Sprintf("%.0f", cfg.MaxSleepInterval),
		"--socket-timeout", fmt.Sprintf("%.0f", cfg.Timeout),
	}

	if impersonate {
		args = append(args, "--impersonate", "chrome")
	}

	// args = append(args, "--cookies-from-browser", "chrome")
	args = append(args, "--cookies", CookiesFile)
	args = append(args, "--", youtubeId)
	return args
}

func handleDownloadErrorNative(info *downloadInfo, youtubeId string, err error, output string) error {
	reason := err.Error()
	if output != "" {
		reason += " | output: " + output
	}

	TrailarrLog(ERROR, "YouTube", "Download failed for %s: %s", youtubeId, reason)
	addToRejectedExtras(info, youtubeId, reason)
	return fmt.Errorf(reason+": %w", err)
}

func addToRejectedExtras(info *downloadInfo, youtubeId, reason string) {
	var rejectedList []RejectedExtra

	_ = ReadJSONFile(RejectedExtrasPath, &rejectedList)

	// Check if already rejected
	for _, rejected := range rejectedList {
		if rejected.YoutubeId == youtubeId {
			return
		}
	}

	rejectedList = append(rejectedList, RejectedExtra{
		MediaType:  info.MediaType,
		MediaId:    info.MediaId,
		MediaTitle: info.MediaTitle,
		ExtraType:  info.ExtraType,
		ExtraTitle: info.ExtraTitle,
		YoutubeId:  youtubeId,
		Reason:     reason,
	})

	// Deduplicate by URL
	tempList := make([]map[string]string, 0, len(rejectedList))
	for _, rejected := range rejectedList {
		tempList = append(tempList, map[string]string{"url": rejected.YoutubeId})
	}
	uniqueURLs := DeduplicateByKey(tempList, "url")
	finalList := make([]RejectedExtra, 0, len(uniqueURLs))
	for _, u := range uniqueURLs {
		for _, rejected := range rejectedList {
			if rejected.YoutubeId == u["url"] {
				finalList = append(finalList, rejected)
				break
			}
		}
	}
	_ = WriteJSONFile(RejectedExtrasPath, finalList)
}

func moveDownloadedFile(info *downloadInfo) error {
	if _, statErr := os.Stat(info.TempFile); statErr != nil {
		TrailarrLog(ERROR, "YouTube", "yt-dlp did not produce expected output file: %s", info.TempFile)
		return fmt.Errorf("yt-dlp did not produce expected output file: %s", info.TempFile)
	}

	if err := os.MkdirAll(info.OutDir, 0755); err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to create output dir '%s': %v", info.OutDir, err)
		return fmt.Errorf("failed to create output dir '%s': %w", info.OutDir, err)
	}

	if moveErr := os.Rename(info.TempFile, info.OutFile); moveErr != nil {
		TrailarrLog(WARN, "YouTube", "Rename failed, attempting cross-device move: %v", moveErr)
		return handleCrossDeviceMove(info.TempFile, info.OutFile, moveErr)
	}

	return nil
}

func handleCrossDeviceMove(tempFile, outFile string, moveErr error) error {
	if linkErr, ok := moveErr.(*os.LinkError); ok && strings.Contains(linkErr.Error(), "cross-device link") {
		TrailarrLog(WARN, "YouTube", "Cross-device link error, copying file instead: %v", moveErr)
		return copyFileAcrossDevices(tempFile, outFile)
	}
	TrailarrLog(ERROR, "YouTube", "Failed to move downloaded file to output dir: %v", moveErr)
	return fmt.Errorf("failed to move downloaded file to output dir: %w", moveErr)
}

func copyFileAcrossDevices(tempFile, outFile string) error {
	in, err := os.Open(tempFile)
	if err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to open temp file for copy: %v", err)
		return fmt.Errorf("failed to open temp file for copy: %w", err)
	}
	defer in.Close()

	out, err := os.Create(outFile)
	if err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to create output file for copy: %v", err)
		return fmt.Errorf("failed to create output file for copy: %w", err)
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to copy file across devices: %v", err)
		return fmt.Errorf("failed to copy file across devices: %w", err)
	}

	if err := out.Sync(); err != nil {
		TrailarrLog(ERROR, "YouTube", "Failed to sync output file: %v", err)
		return fmt.Errorf("failed to sync output file: %w", err)
	}

	if rmErr := os.Remove(tempFile); rmErr != nil {
		TrailarrLog(WARN, "YouTube", "Failed to remove temp file after copy: %v", rmErr)
	}

	return nil
}

func createSuccessMetadata(info *downloadInfo, youtubeId string) (*ExtraDownloadMetadata, error) {
	meta := NewExtraDownloadMetadata(info, youtubeId, "downloaded")

	metaFile := info.OutFile + ".json"
	metaBytes, _ := json.MarshalIndent(meta, "", "  ")
	_ = os.WriteFile(metaFile, metaBytes, 0644)

	TrailarrLog(INFO, "YouTube", "Downloaded %s to %s", info.ExtraTitle, info.OutFile)
	return meta, nil
}
