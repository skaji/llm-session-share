package session

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// CodexTitle reads the latest matching entry in Codex's append-only title index.
// Only read the home containing this rollout under sessions or archived_sessions.
// Missing metadata leaves the UI fallback intact.
func CodexTitle(path, id string) string {
	var home string
	absolute, err := filepath.Abs(path)
	if err == nil {
		for dir := filepath.Dir(absolute); ; dir = filepath.Dir(dir) {
			if filepath.Base(dir) == "sessions" || filepath.Base(dir) == "archived_sessions" {
				home = filepath.Dir(dir)
				break
			}
			if filepath.Dir(dir) == dir {
				break
			}
		}
	}
	if home == "" {
		return ""
	}
	file, err := os.Open(filepath.Join(home, "session_index.jsonl"))
	if err != nil {
		return ""
	}
	defer func() { _ = file.Close() }()
	var title string
	reader := bufio.NewReader(file)
	for {
		line, err := reader.ReadBytes('\n')
		var entry struct {
			ID    string `json:"id"`
			Title string `json:"thread_name"`
		}
		if json.Unmarshal(line, &entry) == nil && entry.ID == id {
			title = strings.TrimSpace(entry.Title)
		}
		if err != nil {
			return title
		}
	}
}
