package share

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCloudflareTokenUploads(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test cloudflared stub uses a POSIX shell")
	}
	dir := t.TempDir()
	stub := `#!/bin/sh
test "$#" -eq 4 && test "$1" = access && test "$2" = token && test "$3" = --app && test "$4" = "$TEST_ACCESS_APP" || exit 2
printf '%s\n' "$TEST_ACCESS_TOKEN"
if test "$TEST_ACCESS_FAIL" = yes; then
  printf '%s\n' "$TEST_ACCESS_TOKEN" >&2
  exit 1
fi
`
	if err := os.WriteFile(filepath.Join(dir, "cloudflared"), []byte(stub), 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	path := filepath.Join(dir, "rollout-"+testID+".jsonl")
	if err := os.WriteFile(path, []byte("{\"role\":\"user\",\"content\":\"hello\"}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	requests := make(chan http.Header, 10)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requests <- r.Header.Clone()
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()
	t.Setenv("TEST_ACCESS_APP", server.URL)
	opts := Options{
		Path: path, Server: server.URL, Name: "skaji", Source: "auto", Interval: time.Second, Once: true, CloudflareAccess: true,
		Headers: http.Header{"Cf-Access-Client-Id": {"service-id"}, "Cf-Access-Client-Secret": {"service-secret"}},
	}
	for _, token := range []string{"first.payload.signature", "renewed.payload.signature"} {
		t.Setenv("TEST_ACCESS_TOKEN", token)
		var logs bytes.Buffer
		if err := Run(context.Background(), opts, &logs, &logs); err != nil {
			t.Fatal(err)
		}
		got := <-requests
		if got.Get("Cf-Access-Token") != token || got.Get("Cf-Access-Client-Id") != "" || got.Get("Cf-Access-Client-Secret") != "" {
			t.Fatal("user token was not sent as the sole Access credential")
		}
		if strings.Contains(logs.String(), token) {
			t.Fatal("token appeared in logs")
		}
	}
	opts.Delete, opts.Name = true, ""
	var deletionLogs bytes.Buffer
	if err := Run(context.Background(), opts, &deletionLogs, &deletionLogs); err != nil {
		t.Fatal(err)
	}
	if got := <-requests; got.Get("Cf-Access-Token") != "renewed.payload.signature" {
		t.Fatal("deletion did not use the cloudflared token")
	}
	if !strings.Contains(deletionLogs.String(), "Deleted:") || strings.Contains(deletionLogs.String(), "renewed.payload.signature") {
		t.Fatal("unexpected deletion log")
	}
	for _, failedCommand := range []bool{false, true} {
		token := "sensitive malformed token"
		t.Setenv("TEST_ACCESS_TOKEN", token)
		if failedCommand {
			t.Setenv("TEST_ACCESS_FAIL", "yes")
		}
		var logs bytes.Buffer
		err := Run(context.Background(), opts, &logs, &logs)
		if err == nil {
			t.Fatal("expected token retrieval failure")
		}
		if strings.Contains(fmt.Sprint(err)+logs.String(), token) {
			t.Fatal("failed command exposed token")
		}
		if len(requests) != 0 {
			t.Fatal("upload attempted without a valid token")
		}
	}
}
