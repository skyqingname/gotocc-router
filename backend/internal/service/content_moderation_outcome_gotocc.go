package service

import (
	"context"
	"errors"
)

var errContentModerationNoEndpoint = errors.New("no content moderation endpoint available")

// 未取得审核结果时保留原因，避免把超时或无可用节点显示成风险判定。
func contentModerationNotReviewedReason(err error) string {
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return "audit_timeout"
	case errors.Is(err, context.Canceled):
		return "audit_canceled"
	case errors.Is(err, errContentModerationNoEndpoint):
		return "audit_no_endpoint"
	default:
		return "audit_dependency_failed"
	}
}
