package service

import "testing"

func TestResolveAdminDirectModel_AdminAliases(t *testing.T) {
	key := &APIKey{User: &User{Role: RoleAdmin}}

	tests := map[string]string{
		"Opus-Latest":        "claude-opus-4-7",
		"opus 4.7":           "claude-opus-4-7",
		"opus_4.7":           "claude-opus-4-7",
		"claude-opus-latest": "claude-opus-4-7",
		"claude-opus-4-7":    "claude-opus-4-7",
	}

	for requested, want := range tests {
		got, ok := ResolveAdminDirectModel(key, requested)
		if !ok {
			t.Fatalf("ResolveAdminDirectModel(%q) ok=false", requested)
		}
		if got != want {
			t.Fatalf("ResolveAdminDirectModel(%q) = %q, want %q", requested, got, want)
		}
	}
}

func TestResolveAdminDirectModel_UserKeysDoNotBypassRouter(t *testing.T) {
	key := &APIKey{User: &User{Role: RoleUser}}

	got, ok := ResolveAdminDirectModel(key, "Opus-Latest")
	if ok {
		t.Fatalf("ResolveAdminDirectModel unexpectedly allowed non-admin key")
	}
	if got != "Opus-Latest" {
		t.Fatalf("ResolveAdminDirectModel changed non-admin model to %q", got)
	}
}

func TestResolveAdminDirectModel_AdminNonDirectModelsStayRouted(t *testing.T) {
	key := &APIKey{User: &User{Role: RoleAdmin}}

	got, ok := ResolveAdminDirectModel(key, "default")
	if ok {
		t.Fatalf("ResolveAdminDirectModel unexpectedly treated default as direct")
	}
	if got != "default" {
		t.Fatalf("ResolveAdminDirectModel changed routed model to %q", got)
	}
}
