package service

import "context"

// ImageTaskHistorySummary counts one user's async image tasks for the task center.
//
// Cost is never attributed to a task. The async worker replays the submit
// request through the ordinary Images handler, and RecordUsage keys the usage
// row by resolveUsageBillingRequestID: "client:" + the submit request's
// X-Client-Request-ID (a fresh UUID from middleware.ClientRequestID), else
// "local:" + its request ID. The task ID is generated independently
// ("imgtask_" + UUID in CreateWithMetadata), async_image_tasks stores no
// request ID and usage_logs stores no task ID, so no exact task-to-charge
// mapping exists. Cost stays 0 with CostAvailable false; it is not estimated
// from time ranges, which would also mix in synchronous image calls.
type ImageTaskHistorySummary struct {
	Total         int64   `json:"total"`
	Completed     int64   `json:"completed"`
	Processing    int64   `json:"processing"`
	Failed        int64   `json:"failed"`
	Cost          float64 `json:"cost"`
	CostAvailable bool    `json:"cost_available"`
}

type ImageTaskHistorySummaryReader interface {
	Summary(ctx context.Context, userID int64, filter ImageTaskHistoryFilter) (*ImageTaskHistorySummary, error)
}

// Summary counts the owner's tasks with the console list filters.
func (s *ImageTaskService) Summary(ctx context.Context, owner ImageTaskOwner, filter ImageTaskHistoryFilter) (*ImageTaskHistorySummary, error) {
	reader, ok := s.history.(ImageTaskHistorySummaryReader)
	if !ok {
		return nil, ErrImageTaskUnavailable
	}
	summary, err := reader.Summary(ctx, owner.UserID, normalizeImageTaskHistoryFilter(filter))
	if err != nil {
		return nil, ErrImageTaskUnavailable.WithCause(err)
	}
	return summary, nil
}
