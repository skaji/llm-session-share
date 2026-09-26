package share

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/skaji/llm-session-share/internal/session"
)

const testID = "01a0de0e-f326-7ec1-bc06-0aee7bbdf319"

func TestWatchRetriesAndOnlySharesConversationChanges(t *testing.T) {
	t.Parallel()
	home := t.TempDir()
	if err := os.Mkdir(filepath.Join(home, "sessions"), 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "sessions", "rollout-2026-09-27T00-00-00-"+testID+".jsonl")
	initial := "{\"role\":\"user\",\"content\":\"Question\"}\n"
	if err := os.WriteFile(path, []byte(initial), 0o600); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var attempts atomic.Int32
	requests := make(chan session.Session, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut || r.URL.Path != "/api/sessions/"+testID || r.Header.Get("CF-Access-Client-Id") != "test-client" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if attempts.Add(1) == 1 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		var snapshot session.Session
		if err := json.NewDecoder(r.Body).Decode(&snapshot); err != nil {
			t.Error(err)
		}
		requests <- snapshot
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	var stdout bytes.Buffer
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, Options{Path: path, Server: server.URL, Name: "skaji", Source: "auto", Interval: 10 * time.Millisecond, Headers: http.Header{"Cf-Access-Client-Id": {"test-client"}}}, &stdout, io.Discard)
	}()
	next := func() session.Session {
		t.Helper()
		select {
		case got := <-requests:
			return got
		case <-time.After(3 * time.Second):
			t.Fatal("timed out waiting for upload")
			return session.Session{}
		}
	}
	if got := next(); got.Name != "skaji" || len(got.Messages) != 1 || attempts.Load() != 2 {
		t.Fatalf("retry failed: %+v", got)
	}
	appendText := func(value string) {
		t.Helper()
		f, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.WriteString(value)
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
	}
	appendText("{\"type\":\"event_msg\",\"payload\":{\"type\":\"token_count\"}}\n{\"role\":\"assistant\",\"content\":\"An")
	select {
	case <-requests:
		t.Fatal("tool metadata or partial text triggered an upload")
	case <-time.After(100 * time.Millisecond):
	}
	appendText("swer\"}\n")
	if got := next(); len(got.Messages) != 2 || got.Messages[1].Text != "Answer" {
		t.Fatalf("append failed: %+v", got)
	}
	index := filepath.Join(home, "session_index.jsonl")
	for _, title := range []string{"Original title", "Renamed title"} {
		f, err := os.OpenFile(index, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
		if err != nil {
			t.Fatal(err)
		}
		err = json.NewEncoder(f).Encode(map[string]string{"id": testID, "thread_name": title})
		_ = f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if got := next(); got.Title != title || len(got.Messages) != 2 {
			t.Fatalf("title-only update failed: %+v", got)
		}
	}
	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}
	if stdout.String() != "Sharing: "+server.URL+"/session/"+testID+"?markdown=1&live=1\n" {
		t.Fatalf("unexpected output: %s", stdout.String())
	}
}

func TestLoginRedirectIsNotUploadSuccess(t *testing.T) {
	t.Parallel()
	path := filepath.Join(t.TempDir(), "rollout-"+testID+".jsonl")
	if err := os.WriteFile(path, []byte("{\"role\":\"user\",\"content\":\"hi\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/login", http.StatusFound)
	}))
	defer server.Close()
	err := Run(context.Background(), Options{Path: path, Server: server.URL, Name: "skaji", Source: "auto", Interval: time.Second, Once: true}, io.Discard, io.Discard)
	if err == nil || !strings.Contains(err.Error(), "302") {
		t.Fatalf("login redirect reported as success: %v", err)
	}
}
