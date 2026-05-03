package claude

import "testing"

func TestNormalizeModelIDStripsClaudeDisplaySuffix(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "strips 1m suffix from current sonnet",
			input:    "claude-sonnet-4-6[1m]",
			expected: "claude-sonnet-4-6",
		},
		{
			name:     "strips suffix with whitespace",
			input:    " claude-sonnet-4-6 [1m] ",
			expected: "claude-sonnet-4-6",
		},
		{
			name:     "strips suffix before legacy override",
			input:    "claude-sonnet-4-5[1m]",
			expected: "claude-sonnet-4-5-20250929",
		},
		{
			name:     "leaves non claude ids untouched",
			input:    "gpt-5.5[high]",
			expected: "gpt-5.5[high]",
		},
		{
			name:     "strips repeated display suffixes defensively",
			input:    "claude-opus-4-7[1m][latest]",
			expected: "claude-opus-4-7",
		},
		{
			name:     "maps stale opus 4.6 client route to latest opus",
			input:    "claude-opus-4-6[1m]",
			expected: "claude-opus-4-7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NormalizeModelID(tt.input); got != tt.expected {
				t.Fatalf("NormalizeModelID(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
