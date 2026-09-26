package session

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCodexTitleOnlyUsesRolloutHome(t *testing.T) {
	home := t.TempDir()
	const id = "01a0de4a-2245-7102-b715-58d2ab09ea8f"
	index := []byte(`{"id":"` + id + `","thread_name":"Local title"}` + "\n")
	if err := os.WriteFile(filepath.Join(home, "session_index.jsonl"), index, 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", home)
	userHome := t.TempDir()
	t.Setenv("HOME", userHome)
	if err := os.Mkdir(filepath.Join(userHome, ".codex"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(userHome, ".codex", "session_index.jsonl"), index, 0o600); err != nil {
		t.Fatal(err)
	}
	for _, directory := range []string{"sessions", "archived_sessions"} {
		if got := CodexTitle(filepath.Join(home, directory, "2026", "rollout.jsonl"), id); got != "Local title" {
			t.Errorf("%s: got %q", directory, got)
		}
	}
	path := filepath.Join(home, "copied", "rollout.jsonl")
	if got := CodexTitle(path, id); got != "" {
		t.Fatalf("unexpected CODEX_HOME fallback: %q", got)
	}
	t.Setenv("CODEX_HOME", "")
	if got := CodexTitle(path, id); got != "" {
		t.Fatalf("unexpected HOME fallback: %q", got)
	}
}
