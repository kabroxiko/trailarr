package internal

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestFetchRootFoldersNon200(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	_, err := FetchRootFolders(ts.URL, "key")
	if err == nil {
		t.Fatalf("expected error for non-200 response")
	}
}

func TestFetchRootFoldersMalformedJSON(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}))
	defer ts.Close()

	_, err := FetchRootFolders(ts.URL, "key")
	if err == nil {
		t.Fatalf("expected error for malformed JSON")
	}
}
