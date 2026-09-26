package session

import (
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// ResolvePath accepts a JSONL path or a local Codex/Claude session link.
func ResolvePath(input string) (string, error) {
	if strings.HasPrefix(input, "claude:") {
		return resolveClaudePath(input)
	}
	if !strings.HasPrefix(input, "codex:") {
		return input, nil
	}
	u, err := url.Parse(input)
	if err != nil || u.Scheme != "codex" || u.Host != "threads" || u.User != nil || u.RawQuery != "" || u.Fragment != "" || !strings.HasPrefix(u.Path, "/") || !validID(strings.TrimPrefix(u.Path, "/")) {
		return "", fmt.Errorf("expected codex://threads/<session-uuid>")
	}
	id := strings.ToLower(strings.TrimPrefix(u.Path, "/"))
	home := os.Getenv("CODEX_HOME")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = filepath.Join(userHome, ".codex")
	}
	var matches []string
	for _, directory := range []string{"sessions", "archived_sessions"} {
		root := filepath.Join(home, directory)
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				if path == root && os.IsNotExist(err) {
					return nil
				}
				return err
			}
			name := strings.ToLower(entry.Name())
			if entry.Type().IsRegular() && strings.HasPrefix(name, "rollout-") && strings.HasSuffix(name, "-"+id+".jsonl") {
				matches = append(matches, path)
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("find Codex session: %w", err)
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no local JSONL for Codex session %s under %s", id, home)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple JSONL files for Codex session %s; specify a file path", id)
	}
	return matches[0], nil
}

func resolveClaudePath(input string) (string, error) {
	u, err := url.Parse(input)
	if err != nil {
		return "", fmt.Errorf("expected claude://resume?session=<session-uuid>")
	}
	query, err := url.ParseQuery(u.RawQuery)
	id := query.Get("session")
	if err != nil || u.Scheme != "claude" || u.Host != "resume" || u.User != nil || u.Path != "" || u.Fragment != "" || len(query) != 1 || len(query["session"]) != 1 || !validID(id) {
		return "", fmt.Errorf("expected claude://resume?session=<session-uuid>")
	}
	home := os.Getenv("CLAUDE_CONFIG_DIR")
	if home == "" {
		userHome, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		home = filepath.Join(userHome, ".claude")
	}
	root := filepath.Join(home, "projects")
	projects, err := os.ReadDir(root)
	if err != nil && !os.IsNotExist(err) {
		return "", err
	}
	var matches []string
	for _, project := range projects {
		if !project.IsDir() {
			continue
		}
		entries, err := os.ReadDir(filepath.Join(root, project.Name()))
		if err != nil {
			return "", err
		}
		for _, entry := range entries {
			if entry.Type().IsRegular() && strings.EqualFold(entry.Name(), id+".jsonl") {
				matches = append(matches, filepath.Join(root, project.Name(), entry.Name()))
			}
		}
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("no local JSONL for Claude session %s under %s", id, home)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple JSONL files for Claude session %s; specify a file path", id)
	}
	return matches[0], nil
}
