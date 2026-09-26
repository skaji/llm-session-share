package share

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDeleteSharedSessionKeepsLocalFile(t *testing.T) {
	t.Parallel()
	for _, source := range []string{"codex", "claude"} {
		t.Run(source, func(t *testing.T) {
			filename := testID + ".jsonl"
			content := `{"type":"user","sessionId":"` + testID + `","message":{"role":"user","content":"original"}}` + "\n"
			if source == "codex" {
				filename = "rollout-" + filename
				content = "{\"role\":\"user\",\"content\":\"original\"}\n"
			}
			path := filepath.Join(t.TempDir(), filename)
			if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
				t.Fatal(err)
			}
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				body, _ := io.ReadAll(r.Body)
				if r.Method != http.MethodDelete || r.URL.Path != "/api/sessions/"+testID || len(body) != 0 || r.Header.Get("CF-Access-Client-Id") != "service-id" {
					t.Error("unexpected deletion request")
				}
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()
			opts := Options{Path: path, Server: server.URL, Source: "auto", Delete: true, Headers: http.Header{"Cf-Access-Client-Id": {"service-id"}}}
			var output bytes.Buffer
			if err := Run(context.Background(), opts, &output, io.Discard); err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(output.String(), "Deleted: "+server.URL+"/session/"+testID) {
				t.Fatalf("unexpected output: %s", output.String())
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != content {
				t.Fatal("local JSONL was modified or removed")
			}
		})
	}
}

func TestDeleteFailureIsReported(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "rollout-"+testID+".jsonl")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusForbidden) }))
	defer server.Close()
	var output bytes.Buffer
	err := Run(context.Background(), Options{Path: path, Server: server.URL, Source: "auto", Delete: true}, &output, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "403") || output.Len() != 0 {
		t.Fatalf("deletion failure not reported: %v", err)
	}
}
