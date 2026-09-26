package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolvePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CODEX_HOME", home)
	const id = "01a0de4a-2245-7102-b715-58d2ab09ea8f"
	const link = "codex://threads/" + id
	if got, err := ResolvePath("copied.jsonl"); err != nil || got != "copied.jsonl" {
		t.Fatalf("plain path: %q, %v", got, err)
	}
	for _, input := range []string{link, "codex://threads/invalid", link + "/extra", "codex://other/" + id, link + "?x=1"} {
		if _, err := ResolvePath(input); err == nil {
			t.Fatalf("expected error for %s", input)
		}
	}
	write := func(directory string) string {
		t.Helper()
		path := filepath.Join(home, directory, "2026", "09", "27", "rollout-2026-09-27T00-16-35-"+id+".jsonl")
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, nil, 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	active := write("sessions")
	if got, err := ResolvePath(link); err != nil || got != active {
		t.Fatalf("active: %q, %v", got, err)
	}
	archived := write("archived_sessions")
	if _, err := ResolvePath(link); err == nil {
		t.Fatal("expected ambiguity error")
	}
	if err := os.Remove(active); err != nil {
		t.Fatal(err)
	}
	if got, err := ResolvePath(link); err != nil || got != archived {
		t.Fatalf("archived: %q, %v", got, err)
	}
	// The default home resolves links too, without requiring CODEX_HOME.
	userHome := t.TempDir()
	if err := os.Rename(home, filepath.Join(userHome, ".codex")); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", userHome)
	t.Setenv("CODEX_HOME", "")
	if _, err := ResolvePath(link); err != nil {
		t.Fatal(err)
	}
}

func TestResolveClaudePath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", home)
	const id = "01a0de4a-2245-7102-b715-58d2ab09ea8f"
	const link = "claude://resume?session=" + id
	for _, input := range []string{link, "claude://resume?session=no", link + "&session=" + id, "claude://other?session=" + id} {
		if _, err := ResolvePath(input); err == nil {
			t.Fatalf("expected error: %s", input)
		}
	}
	path := filepath.Join(home, "projects", "project", id+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if got, err := ResolvePath(link); err != nil || got != path {
		t.Fatalf("got %q, %v", got, err)
	}
	other := filepath.Join(home, "projects", "other", id+".jsonl")
	if err := os.MkdirAll(filepath.Dir(other), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(other, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolvePath(link); err == nil {
		t.Fatal("expected ambiguity")
	}
}
