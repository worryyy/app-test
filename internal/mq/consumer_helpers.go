package mq

import "strings"

func tokenizeText(text string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return ""
	}
	parts := strings.Fields(trimmed)
	if len(parts) == 0 {
		return trimmed
	}
	return strings.Join(parts, " ")
}

func pickFiltered(origin, filtered string) string {
	if strings.TrimSpace(filtered) == "" {
		return origin
	}
	return filtered
}

func isRisky(suggest string) bool {
	v := strings.ToLower(strings.TrimSpace(suggest))
	return v == "risky" || v == "block"
}
