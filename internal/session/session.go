package session

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strings"

	shared "github.com/skaji/llm-session-internal"
)

type Message struct {
	Role      string `json:"role"`
	Text      string `json:"text"`
	Timestamp string `json:"timestamp,omitempty"`
}

type Session struct {
	Title    string    `json:"title,omitempty"`
	ID       string    `json:"id"`
	Source   string    `json:"source"`
	Name     string    `json:"name"`
	Messages []Message `json:"messages"`
}

var uuidPattern = regexp.MustCompile(`(?i)[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)

// Read builds a snapshot from complete records. A partially written final record
// is retried on the next read. Unknown records are ignored, just as in search.
func Read(reader io.Reader, path, name, source string) (Session, error) {
	result := Session{Name: name, Source: source, Messages: []Message{}}
	decoded, err := shared.Decode(reader, path)
	if err != nil {
		return result, err
	}
	defer func() { _ = decoded.Close() }()
	reader = decoded
	result.ID = strings.ToLower(uuidPattern.FindString(filepath.Base(path)))
	if result.Source == "auto" {
		result.Source = ""
	}
	if result.Source == "" && strings.HasPrefix(filepath.Base(path), "rollout-") {
		result.Source = "codex"
	}
	buffer := bufio.NewReader(reader)
	for {
		line, readErr := buffer.ReadBytes('\n')
		var root map[string]any
		if json.Unmarshal(line, &root) == nil && root != nil {
			payload, _ := root["payload"].(map[string]any)
			if root["type"] == "session_meta" {
				if result.Source == "" {
					result.Source = "codex"
				}
				if id, _ := payload["id"].(string); validID(id) {
					result.ID = strings.ToLower(id)
				}
			}
			if id, _ := root["sessionId"].(string); validID(id) {
				result.ID = strings.ToLower(id)
				if result.Source == "" {
					result.Source = "claude"
				}
			}
			if result.Source == "" && (root["type"] == "response_item" || root["role"] != nil) {
				result.Source = "codex"
			}
			if result.Source == "" && (root["type"] == "custom-title" || root["type"] == "ai-title") {
				result.Source = "claude"
			}
			record := shared.Extract(result.Source, line)
			text, role := record.Text, record.Role
			if record.Title != "" && (record.ForceTitle || result.Title == "") {
				result.Title = record.Title
			}

			if text != "" && (role == "user" || role == "assistant") {
				timestamp, _ := root["timestamp"].(string)
				result.Messages = append(result.Messages, Message{Role: role, Text: text, Timestamp: timestamp})
			}
		}
		if readErr != nil {
			if !errors.Is(readErr, io.EOF) {
				return result, readErr
			}
			break
		}
	}
	if result.ID == "" {
		return result, fmt.Errorf("no session UUID found in %s", path)
	}
	if result.Source != "codex" && result.Source != "claude" {
		return result, fmt.Errorf("cannot detect session format; use -source codex or -source claude")
	}
	return result, nil
}

func validID(id string) bool {
	return shared.ValidID(id)
}
