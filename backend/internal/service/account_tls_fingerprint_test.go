package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestEnsureAnthropicOAuthTLSFingerprintEnabled(t *testing.T) {
	tests := []struct {
		name      string
		account   *Account
		wantExtra map[string]any
	}{
		{
			name: "Anthropic OAuth gets TLS enabled even when extra is nil",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeOAuth,
			},
			wantExtra: map[string]any{"enable_tls_fingerprint": true},
		},
		{
			name: "Anthropic setup-token gets TLS enabled",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeSetupToken,
				Extra:    map[string]any{"existing": "value"},
			},
			wantExtra: map[string]any{"existing": "value", "enable_tls_fingerprint": true},
		},
		{
			name: "Anthropic OAuth explicit false is forced back on",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeOAuth,
				Extra:    map[string]any{"enable_tls_fingerprint": false},
			},
			wantExtra: map[string]any{"enable_tls_fingerprint": true},
		},
		{
			name: "Anthropic API key is left alone",
			account: &Account{
				Platform: PlatformAnthropic,
				Type:     AccountTypeAPIKey,
				Extra:    map[string]any{},
			},
			wantExtra: map[string]any{},
		},
		{
			name: "OpenAI OAuth is left alone",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra:    map[string]any{},
			},
			wantExtra: map[string]any{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			EnsureAnthropicOAuthTLSFingerprintEnabled(tt.account)
			require.Equal(t, tt.wantExtra, tt.account.Extra)
		})
	}
}
