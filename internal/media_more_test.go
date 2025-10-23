package internal

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

const textPlain = "text/plain"

func TestFetchFirstSuccessfulFailure(t *testing.T) {
	s1 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer s1.Close()
	s2 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	}))
	defer s2.Close()
	if _, err := fetchFirstSuccessful([]string{s1.URL, s2.URL}); err == nil {
		t.Fatalf("expected error when no successful responses")
	}
}

func TestFetchAndCachePosterFailureStatus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	tmp := t.TempDir()
	local := filepath.Join(tmp, "out.png")
	if err := fetchAndCachePoster(local, srv.URL, "test"); err == nil {
		t.Fatalf("expected error when poster server returns non-200")
	}
}

func TestServeCachedFileAndStreamResponse(t *testing.T) {
	// prepare recorder and gin context
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	// test serveCachedFile GET
	req := httptest.NewRequest("GET", "/", nil)
	c.Request = req
	// create temp file
	tmp := t.TempDir()
	fpath := filepath.Join(tmp, "f.txt")
	_ = os.WriteFile(fpath, []byte("hello"), 0o644)
	serveCachedFile(c, fpath, textPlain)
	if w.Code != http.StatusOK {
		t.Fatalf("serveCachedFile GET returned %d", w.Code)
	}
	if w.Header().Get(HeaderContentType) != textPlain {
		t.Fatalf("serveCachedFile content-type not set: %s", w.Header().Get(HeaderContentType))
	}

	// test serveCachedFile HEAD
	w2 := httptest.NewRecorder()
	c2, _ := gin.CreateTestContext(w2)
	req2 := httptest.NewRequest("HEAD", "/", nil)
	c2.Request = req2
	serveCachedFile(c2, fpath, textPlain)
	if w2.Code != http.StatusOK {
		t.Fatalf("serveCachedFile HEAD returned %d", w2.Code)
	}

	// test streamResponse writes body
	w3 := httptest.NewRecorder()
	c3, _ := gin.CreateTestContext(w3)
	req3 := httptest.NewRequest("GET", "/", nil)
	c3.Request = req3
	streamResponse(c3, textPlain, strings.NewReader("streamed"))
	if w3.Code != http.StatusOK {
		t.Fatalf("streamResponse returned %d", w3.Code)
	}
	if strings.TrimSpace(w3.Body.String()) != "streamed" {
		t.Fatalf("streamResponse body mismatch: %s", w3.Body.String())
	}
}

func TestCachedYouTubeImagePicksCorrectExt(t *testing.T) {
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "YouTube")
	_ = os.MkdirAll(dir, 0o755)
	// create webp file
	_ = os.WriteFile(filepath.Join(dir, "y.webp"), []byte("x"), 0o644)
	p, ct := cachedYouTubeImage(dir, "y")
	if !strings.HasSuffix(p, "y.webp") || ct != "image/webp" {
		t.Fatalf("cachedYouTubeImage did not choose webp: %s %s", p, ct)
	}
}
