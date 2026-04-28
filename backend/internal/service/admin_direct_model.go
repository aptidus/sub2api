package service

import "strings"

const AdminDirectOpusLatestModel = "claude-opus-4-7"

// ResolveAdminDirectModel canonicalizes admin-only direct model aliases.
// Normal user keys must keep using the public routed model surface.
func ResolveAdminDirectModel(apiKey *APIKey, requestedModel string) (string, bool) {
	if apiKey == nil || apiKey.User == nil || !apiKey.User.IsAdmin() {
		return requestedModel, false
	}

	normalized := strings.ToLower(strings.TrimSpace(requestedModel))
	normalized = strings.ReplaceAll(normalized, "_", "-")
	normalized = strings.Join(strings.Fields(normalized), "-")

	switch normalized {
	case "opus-latest", "opus-4.7", "opus4.7", "claude-opus-latest":
		return AdminDirectOpusLatestModel, true
	case "claude-opus-4-7":
		return normalized, true
	default:
		return requestedModel, false
	}
}
