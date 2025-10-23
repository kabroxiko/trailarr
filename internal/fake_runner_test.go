package internal

import (
	"bytes"
	"context"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

// fakeRunner implements YtDlpRunner for tests.
type fakeRunner struct{}

const testYtID = "yt-123"

func (f *fakeRunner) StartCommand(ctx context.Context, name string, args []string) (io.ReadCloser, *exec.Cmd, error) {
	// produce two JSON lines and then EOF
	lines := []string{
		`{"id":"vid1","title":"t1","thumbnail":"th1"}` + "\n",
		`{"id":"vid2","title":"t2","thumbnail":"th2"}` + "\n",
	}
	r := bytes.NewBufferString(strings.Join(lines, ""))
	// return a simple ReadCloser; the cmd is a placeholder (Wait is ignored in callers)
	return io.NopCloser(r), &exec.Cmd{}, nil
}

func (f *fakeRunner) CombinedOutput(name string, args []string, dir string) ([]byte, error) {
	// Attempt to locate --output arg to create the temp file
	outPath := ""
	for i := 0; i < len(args)-1; i++ {
		if args[i] == "--output" {
			outPath = args[i+1]
			break
		}
	}
	if outPath == "" {
		// fallback: create a file in dir
		outPath = filepath.Join(dir, "fake-output.mkv")
	} else if !filepath.IsAbs(outPath) && dir != "" {
		outPath = filepath.Join(dir, outPath)
	}
	// ensure dir exists
	_ = os.MkdirAll(filepath.Dir(outPath), 0o755)
	_ = os.WriteFile(outPath, []byte("dummy"), 0o644)
	// return a fake stdout string
	return []byte("[info] download complete\n"), nil
}

func TestRunYtDlpSearchRealWithFakeRunner(t *testing.T) {
	old := ytDlpRunner
	ytDlpRunner = &fakeRunner{}
	defer func() { ytDlpRunner = old }()

	// run runYtDlpSearchReal which reads from StartCommand and parses lines
	results := &[]gin.H{}
	vidSet := map[string]bool{}
	// give a small timeout context
	err := runYtDlpSearchReal("term trailer", vidSet, results, 10, []string{"-j", "ytsearch:term trailer", "--skip-download"})
	if err != nil {
		t.Fatalf("runYtDlpSearchReal returned error: %v", err)
	}
	if len(*results) != 2 {
		t.Fatalf("expected 2 results from fake runner, got %d", len(*results))
	}
}

func TestPerformDownloadWithFakeRunner(t *testing.T) {
	oldRunner := ytDlpRunner
	ytDlpRunner = &fakeRunner{}
	defer func() { ytDlpRunner = oldRunner }()

	// prepare a downloadInfo via prepareDownloadInfo but override temp dir to temp test dir
	info, err := prepareDownloadInfo("movie", 1, "Trailer", "T", testYtID)
	if err != nil {
		t.Fatalf("prepareDownloadInfo failed: %v", err)
	}
	// Ensure temp dir exists
	_ = os.MkdirAll(info.TempDir, 0o755)

	// Call performDownload which should call CombinedOutput and then move file
	meta, err := performDownload(info, testYtID)
	if err != nil {
		t.Fatalf("performDownload failed: %v", err)
	}
	if meta == nil || meta.YouTubeID != testYtID {
		t.Fatalf("unexpected metadata: %+v", meta)
	}
	// output file should exist at OutFile
	// allow a small wait for filesystem ops
	time.Sleep(10 * time.Millisecond)
	if _, err := os.Stat(info.OutFile); err != nil {
		t.Fatalf("expected output file at %s, stat error: %v", info.OutFile, err)
	}
}
