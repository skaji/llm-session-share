package session

import "strings"

// Conversation filtering follows llm-session-search/internal/search/codex.go.
func codexConversationText(value any) (string, string) {
	root, ok := value.(map[string]any)
	if !ok {
		return "", ""
	}
	message := root
	if payload, ok := root["payload"].(map[string]any); ok {
		if role, _ := payload["role"].(string); role == "user" || role == "assistant" {
			message = payload
		}
	}
	role, _ := message["role"].(string)
	if role != "user" && role != "assistant" {
		return "", ""
	}

	var parts []string
	for _, key := range []string{"message", "content", "text"} {
		appendCodexConversationValue(&parts, message[key])
	}
	if len(parts) == 0 {
		appendCodexConversationValue(&parts, root["message"])
	}
	return strings.Join(parts, "\n"), role
}

func appendCodexConversationValue(parts *[]string, value any) {
	switch value := value.(type) {
	case string:
		value = strings.TrimSpace(value)
		if value != "" && !isBase64DataURL(value) {
			*parts = append(*parts, value)
		}
	case map[string]any:
		for _, key := range []string{"message", "content", "text"} {
			appendCodexConversationValue(parts, value[key])
		}
	case []any:
		for _, child := range value {
			appendCodexConversationValue(parts, child)
		}
	}
}

func sanitizeCodexInjectedContext(value any) {
	switch value := value.(type) {
	case map[string]any:
		filterCodexInjectedContent(value)
		for _, child := range value {
			sanitizeCodexInjectedContext(child)
		}
	case []any:
		for _, child := range value {
			sanitizeCodexInjectedContext(child)
		}
	}
}

func filterCodexInjectedContent(object map[string]any) {
	content, ok := object["content"].([]any)
	if !ok {
		return
	}
	metadata, _ := object["internal_chat_message_metadata_passthrough"].(map[string]any)
	kinds, _ := metadata["content_item_kinds"].([]any)
	kindsAlign := len(content) == len(kinds)

	filtered := make([]any, 0, len(content))
	for index, item := range content {
		if kindsAlign {
			kind, _ := kinds[index].(string)
			if !isCodexInjectedContentKind(kind) {
				filtered = append(filtered, item)
			}
		} else if !isLegacyCodexInjectedContentItem(item) {
			filtered = append(filtered, item)
		}
	}
	object["content"] = filtered
	delete(object, "internal_chat_message_metadata_passthrough")
}

func isLegacyCodexInjectedContentItem(item any) bool {
	object, ok := item.(map[string]any)
	if !ok {
		return false
	}
	text, ok := object["text"].(string)
	if !ok {
		return false
	}
	text = strings.TrimSpace(text)
	return strings.HasPrefix(text, "# AGENTS.md instructions for ") && strings.Contains(text, "<INSTRUCTIONS>") ||
		strings.HasPrefix(text, "<recommended_plugins>") && strings.Contains(text, "</recommended_plugins>") ||
		strings.HasPrefix(text, "<environment_context>") && strings.Contains(text, "</environment_context>")
}

func isCodexInjectedContentKind(kind string) bool {
	switch kind {
	case "agents_md.instructions", "plugins.recommendations", "environments.environment_context":
		return true
	default:
		return false
	}
}
