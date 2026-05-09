//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var _ OpsRepository = (*stubOpsRepo)(nil)

type stubOpsRepo struct {
	OpsRepository
	overview *OpsDashboardOverview
	err      error
}

func (s *stubOpsRepo) GetDashboardOverview(ctx context.Context, filter *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	if s.err != nil {
		return nil, s.err
	}
	if s.overview != nil {
		return s.overview, nil
	}
	return &OpsDashboardOverview{}, nil
}

type trafficShapeAccountRepoStub struct {
	AccountRepository
	accounts []Account
}

func (s *trafficShapeAccountRepoStub) ListByPlatform(ctx context.Context, platform string) ([]Account, error) {
	out := make([]Account, 0, len(s.accounts))
	for _, account := range s.accounts {
		if account.Platform == platform {
			out = append(out, account)
		}
	}
	return out, nil
}

type trafficShapeUsageRepoStub struct {
	UsageLogRepository
	fiveMinute map[int64]AccountRiskWindowStats
	fiveHour   map[int64]AccountRiskWindowStats
}

func (s *trafficShapeUsageRepoStub) GetAccountRiskWindowStatsBatch(ctx context.Context, accountIDs []int64, startTime, endTime time.Time) (map[int64]AccountRiskWindowStats, error) {
	out := make(map[int64]AccountRiskWindowStats, len(accountIDs))
	window := s.fiveHour
	if endTime.Sub(startTime) <= 10*time.Minute {
		window = s.fiveMinute
	}
	for _, accountID := range accountIDs {
		if stat, ok := window[accountID]; ok {
			out[accountID] = stat
		} else {
			out[accountID] = AccountRiskWindowStats{AccountID: accountID}
		}
	}
	return out, nil
}

func (s *trafficShapeUsageRepoStub) GetAccountRiskDimensionStats(ctx context.Context, accountIDs []int64, startTime, endTime time.Time, dimension string, limit int) (map[int64][]AccountRiskDimensionStat, error) {
	return map[int64][]AccountRiskDimensionStat{}, nil
}

func TestComputeGroupAvailableRatio(t *testing.T) {
	t.Parallel()

	t.Run("正常情况: 10个账号, 8个可用 = 80%", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  10,
			AvailableCount: 8,
		})
		require.InDelta(t, 80.0, got, 0.0001)
	})

	t.Run("边界情况: TotalAccounts = 0 应返回 0", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  0,
			AvailableCount: 8,
		})
		require.Equal(t, 0.0, got)
	})

	t.Run("边界情况: AvailableCount = 0 应返回 0%", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  10,
			AvailableCount: 0,
		})
		require.Equal(t, 0.0, got)
	})
}

func TestCountAccountsByCondition(t *testing.T) {
	t.Parallel()

	t.Run("测试限流账号统计: acc.IsRateLimited", func(t *testing.T) {
		t.Parallel()

		accounts := map[int64]*AccountAvailability{
			1: {IsRateLimited: true},
			2: {IsRateLimited: false},
			3: {IsRateLimited: true},
		}

		got := countAccountsByCondition(accounts, func(acc *AccountAvailability) bool {
			return acc.IsRateLimited
		})
		require.Equal(t, int64(2), got)
	})

	t.Run("测试错误账号统计（排除临时不可调度）: acc.HasError && acc.TempUnschedulableUntil == nil", func(t *testing.T) {
		t.Parallel()

		until := time.Now().UTC().Add(5 * time.Minute)
		accounts := map[int64]*AccountAvailability{
			1: {HasError: true},
			2: {HasError: true, TempUnschedulableUntil: &until},
			3: {HasError: false},
		}

		got := countAccountsByCondition(accounts, func(acc *AccountAvailability) bool {
			return acc.HasError && acc.TempUnschedulableUntil == nil
		})
		require.Equal(t, int64(1), got)
	})

	t.Run("边界情况: 空 map 应返回 0", func(t *testing.T) {
		t.Parallel()

		got := countAccountsByCondition(map[int64]*AccountAvailability{}, func(acc *AccountAvailability) bool {
			return acc.IsRateLimited
		})
		require.Equal(t, int64(0), got)
	})
}

func TestComputeRuleMetricNewIndicators(t *testing.T) {
	t.Parallel()

	groupID := int64(101)
	platform := "openai"

	availability := &OpsAccountAvailability{
		Group: &GroupAvailability{
			GroupID:        groupID,
			TotalAccounts:  10,
			AvailableCount: 8,
		},
		Accounts: map[int64]*AccountAvailability{
			1: {IsRateLimited: true},
			2: {IsRateLimited: true},
			3: {HasError: true},
			4: {HasError: true, TempUnschedulableUntil: timePtr(time.Now().UTC().Add(2 * time.Minute))},
			5: {HasError: false, IsRateLimited: false},
		},
	}

	opsService := &OpsService{
		getAccountAvailability: func(_ context.Context, _ string, _ *int64) (*OpsAccountAvailability, error) {
			return availability, nil
		},
	}

	svc := &OpsAlertEvaluatorService{
		opsService: opsService,
		opsRepo:    &stubOpsRepo{overview: &OpsDashboardOverview{}},
	}

	start := time.Now().UTC().Add(-5 * time.Minute)
	end := time.Now().UTC()
	ctx := context.Background()

	tests := []struct {
		name       string
		metricType string
		groupID    *int64
		wantValue  float64
		wantOK     bool
	}{
		{
			name:       "group_available_accounts",
			metricType: "group_available_accounts",
			groupID:    &groupID,
			wantValue:  8,
			wantOK:     true,
		},
		{
			name:       "group_available_ratio",
			metricType: "group_available_ratio",
			groupID:    &groupID,
			wantValue:  80.0,
			wantOK:     true,
		},
		{
			name:       "account_rate_limited_count",
			metricType: "account_rate_limited_count",
			groupID:    nil,
			wantValue:  2,
			wantOK:     true,
		},
		{
			name:       "account_error_count",
			metricType: "account_error_count",
			groupID:    nil,
			wantValue:  1,
			wantOK:     true,
		},
		{
			name:       "group_available_accounts without group_id returns false",
			metricType: "group_available_accounts",
			groupID:    nil,
			wantValue:  0,
			wantOK:     false,
		},
		{
			name:       "group_available_ratio without group_id returns false",
			metricType: "group_available_ratio",
			groupID:    nil,
			wantValue:  0,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rule := &OpsAlertRule{
				MetricType: tt.metricType,
			}
			gotValue, gotOK := svc.computeRuleMetric(ctx, rule, nil, start, end, platform, tt.groupID)
			require.Equal(t, tt.wantOK, gotOK)
			if !tt.wantOK {
				return
			}
			require.InDelta(t, tt.wantValue, gotValue, 0.0001)
		})
	}
}

func TestComputeRuleMetricTrafficShapeIndicators(t *testing.T) {
	t.Parallel()

	groupID := int64(101)
	accountRepo := &trafficShapeAccountRepoStub{
		accounts: []Account{
			{
				ID:       1,
				Platform: PlatformAnthropic,
				Type:     AccountTypeOAuth,
				Groups:   []*Group{{ID: groupID}},
				Extra: map[string]any{
					"risk_max_requests_5m": 100,
				},
			},
			{
				ID:       2,
				Platform: PlatformAnthropic,
				Type:     AccountTypeOAuth,
				Groups:   []*Group{{ID: groupID}},
				Extra: map[string]any{
					"risk_max_requests_5m": 100,
				},
			},
			{
				ID:       3,
				Platform: PlatformAnthropic,
				Type:     AccountTypeOAuth,
				Groups:   []*Group{{ID: 202}},
				Extra: map[string]any{
					"risk_max_requests_5m": 100,
				},
			},
		},
	}
	usageRepo := &trafficShapeUsageRepoStub{
		fiveMinute: map[int64]AccountRiskWindowStats{
			1: {AccountID: 1, Requests: 70},
			2: {AccountID: 2, Requests: 90},
			3: {AccountID: 3, Requests: 100},
		},
		fiveHour: map[int64]AccountRiskWindowStats{},
	}
	svc := &OpsAlertEvaluatorService{
		opsRepo: &stubOpsRepo{overview: &OpsDashboardOverview{}},
		accountUsageService: &AccountUsageService{
			accountRepo:  accountRepo,
			usageLogRepo: usageRepo,
		},
	}

	start := time.Now().UTC().Add(-5 * time.Minute)
	end := time.Now().UTC()
	ctx := context.Background()

	tests := []struct {
		name       string
		metricType string
		groupID    *int64
		wantValue  float64
	}{
		{
			name:       "max score all groups",
			metricType: "account_traffic_shape_max_score",
			wantValue:  100,
		},
		{
			name:       "hot count scoped group",
			metricType: "account_traffic_shape_hot_count",
			groupID:    &groupID,
			wantValue:  2,
		},
		{
			name:       "sticky-only count scoped group",
			metricType: "account_traffic_shape_sticky_only_count",
			groupID:    &groupID,
			wantValue:  1,
		},
		{
			name:       "hard-cap count all groups",
			metricType: "account_traffic_shape_hard_cap_count",
			wantValue:  1,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotValue, gotOK := svc.computeRuleMetric(ctx, &OpsAlertRule{MetricType: tt.metricType}, nil, start, end, PlatformAnthropic, tt.groupID)
			require.True(t, gotOK)
			require.InDelta(t, tt.wantValue, gotValue, 0.0001)
		})
	}
}
