package session

import "strings"

// Conversation filtering follows llm-session-search/internal/search/claude.go.
func claudeConversationText(content any) string {
	if text, ok := content.(string); ok {
		text = cleanClaudeConversationText(text)
		if !isBase64DataURL(text) {
			return text
		}
		return ""
	}

	items, ok := content.([]any)
	if !ok {
		return ""
	}
	var parts []string
	for _, item := range items {
		object, ok := item.(map[string]any)
		if !ok || object["type"] != "text" {
			continue
		}
		text, _ := object["text"].(string)
		text = cleanClaudeConversationText(text)
		if text != "" && !isBase64DataURL(text) {
			parts = append(parts, text)
		}
	}
	return strings.Join(parts, "\n")
}

func cleanClaudeConversationText(text string) string {
	text = strings.TrimSpace(text)
	for _, prefix := range []string{
		"<local-command-caveat>",
		"<command-name>",
		"<command-message>",
		"<command-args>",
		"<local-command-stdout>",
		"<task-notification>",
	} {
		if strings.HasPrefix(text, prefix) {
			return ""
		}
	}
	for {
		removed := false
		for _, tag := range []string{
			"system-reminder",
			"ide_opened_file",
			"ide_selection",
		} {
			opening := "<" + tag + ">"
			if !strings.HasPrefix(text, opening) {
				continue
			}
			closing := "</" + tag + ">"
			end := strings.Index(text, closing)
			if end < 0 {
				return ""
			}
			text = strings.TrimSpace(text[end+len(closing):])
			removed = true
			break
		}
		if !removed {
			return text
		}
	}
}
