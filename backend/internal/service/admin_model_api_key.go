package service

import (
	"context"
	"fmt"
)

const AdminModelAPIKeyName = "admin-direct-model"

// BuildAdminModelAPIKeyContext creates the synthetic API-key context used when
// the global admin API key is intentionally used for model calls. It is not a
// persisted user key and does not bypass upstream account selection.
func (s *APIKeyService) BuildAdminModelAPIKeyContext(ctx context.Context) (*APIKey, error) {
	if s == nil || s.userRepo == nil || s.groupRepo == nil {
		return nil, fmt.Errorf("admin model API key context is not available")
	}

	admin, err := s.userRepo.GetFirstAdmin(ctx)
	if err != nil {
		return nil, fmt.Errorf("get first admin: %w", err)
	}
	if admin == nil || !admin.IsActive() || !admin.IsAdmin() {
		return nil, fmt.Errorf("active admin user not found")
	}

	groups, err := s.groupRepo.ListActiveByPlatform(ctx, PlatformAnthropic)
	if err != nil {
		return nil, fmt.Errorf("list active anthropic groups: %w", err)
	}
	if len(groups) == 0 {
		return nil, fmt.Errorf("active anthropic group not found")
	}

	group := groups[0]
	return &APIKey{
		ID:     0,
		UserID: admin.ID,
		Key:    AdminModelAPIKeyName,
		Name:   AdminModelAPIKeyName,
		Status: StatusAPIKeyActive,
		GroupID: func() *int64 {
			id := group.ID
			return &id
		}(),
		User:  admin,
		Group: &group,
	}, nil
}

func IsSyntheticAdminModelAPIKey(apiKey *APIKey) bool {
	return apiKey != nil && apiKey.ID == 0 && apiKey.Name == AdminModelAPIKeyName
}
