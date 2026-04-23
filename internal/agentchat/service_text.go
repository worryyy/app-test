package agentchat

import "strings"

func buildConversationTitle(content string) string {
	title := previewText(content, 32)
	if title == "" {
		return "新对话"
	}
	return title
}

func previewText(content string, limit int) string {
	content = strings.TrimSpace(content)
	if content == "" || limit <= 0 {
		return ""
	}

	runes := []rune(content)
	if len(runes) <= limit {
		return content
	}
	return strings.TrimSpace(string(runes[:limit])) + "..."
}
