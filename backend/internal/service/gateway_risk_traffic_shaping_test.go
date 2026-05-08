package service

import "testing"

func TestEvaluateAccountRiskSchedulability(t *testing.T) {
	account := &Account{
		ID:       1,
		Platform: PlatformAnthropic,
		Type:     AccountTypeOAuth,
		Extra: map[string]any{
			"risk_max_requests_5m":          100,
			"risk_max_cache_read_tokens_5m": 1000,
			"risk_max_total_tokens_5h":      10000,
			"risk_max_distinct_users_5m":    10,
			"risk_max_distinct_ips_5m":      10,
			"risk_throttle_ratio":           0.70,
			"risk_sticky_only_ratio":        0.85,
			"risk_hard_cap_ratio":           1.00,
		},
	}

	t.Run("below sticky threshold remains schedulable", func(t *testing.T) {
		eval := evaluateAccountRiskSchedulability(account, AccountRiskWindowStats{
			AccountID: 1,
			Requests:  75,
		}, AccountRiskWindowStats{AccountID: 1})

		if eval.schedulability != WindowCostSchedulable {
			t.Fatalf("schedulability = %v, want schedulable", eval.schedulability)
		}
		if !riskEvaluationAllows(eval, false) {
			t.Fatal("non-sticky request should still be allowed")
		}
	})

	t.Run("hot account becomes sticky only", func(t *testing.T) {
		eval := evaluateAccountRiskSchedulability(account, AccountRiskWindowStats{
			AccountID: 1,
			Requests:  90,
		}, AccountRiskWindowStats{AccountID: 1})

		if eval.schedulability != WindowCostStickyOnly {
			t.Fatalf("schedulability = %v, want sticky-only", eval.schedulability)
		}
		if riskEvaluationAllows(eval, false) {
			t.Fatal("non-sticky request should be blocked")
		}
		if !riskEvaluationAllows(eval, true) {
			t.Fatal("sticky request should still be allowed")
		}
	})

	t.Run("hard cap blocks even sticky traffic", func(t *testing.T) {
		eval := evaluateAccountRiskSchedulability(account, AccountRiskWindowStats{
			AccountID:       1,
			CacheReadTokens: 1000,
		}, AccountRiskWindowStats{AccountID: 1})

		if eval.schedulability != WindowCostNotSchedulable {
			t.Fatalf("schedulability = %v, want not schedulable", eval.schedulability)
		}
		if riskEvaluationAllows(eval, true) {
			t.Fatal("hard-capped account should be blocked")
		}
	})

	t.Run("display limits use same threshold math", func(t *testing.T) {
		limits := buildAccountTrafficShapeLimits(account, AccountRiskWindowStats{
			AccountID:       1,
			Requests:        70,
			CacheReadTokens: 900,
		}, AccountRiskWindowStats{
			AccountID: 1,
			Tokens:    10000,
		})

		byName := make(map[string]AccountTrafficShapeLimit, len(limits))
		for _, limit := range limits {
			byName[limit.Name] = limit
		}
		if byName["requests_5m"].State != "throttled" {
			t.Fatalf("requests_5m state = %q, want throttled", byName["requests_5m"].State)
		}
		if byName["cache_read_5m"].State != "sticky_only" {
			t.Fatalf("cache_read_5m state = %q, want sticky_only", byName["cache_read_5m"].State)
		}
		if byName["tokens_5h"].State != "hard_cap" {
			t.Fatalf("tokens_5h state = %q, want hard_cap", byName["tokens_5h"].State)
		}
	})
}
