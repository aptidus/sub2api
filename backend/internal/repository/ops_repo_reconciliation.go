package repository

import (
	"context"
	"fmt"
	"time"
)

func (r *opsRepository) CountUsageBillingMissingLogs(ctx context.Context, start, end time.Time) (int64, error) {
	if r == nil || r.db == nil {
		return 0, fmt.Errorf("nil ops repository")
	}
	if end.IsZero() {
		end = time.Now().UTC()
	}
	if start.IsZero() || !start.Before(end) {
		start = end.Add(-5 * time.Minute)
	}

	var count int64
	err := r.db.QueryRowContext(ctx, `
SELECT COUNT(*)
FROM usage_billing_dedup d
LEFT JOIN usage_logs l
  ON l.request_id = d.request_id
 AND l.api_key_id = d.api_key_id
WHERE d.created_at >= $1
  AND d.created_at < $2
  AND l.id IS NULL
`, start, end).Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}
