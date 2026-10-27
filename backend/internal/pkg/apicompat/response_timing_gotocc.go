package apicompat

// ChatResponseTiming keeps the gateway's measured wall time separate from
// the response lifecycle declared by the immediate upstream supplier.
type ChatResponseTiming struct {
	ForwardDurationMs int64                   `json:"forward_duration_ms"`
	UpstreamResponse  *ReportedResponseTiming `json:"upstream_response,omitempty"`
}

// ReportedResponseTiming retains the timestamps used for the calculation.
// Timestamp values are seconds since the Unix epoch. Millisecond units on
// DurationMs do not imply that the supplier supplied millisecond precision.
type ReportedResponseTiming struct {
	CreatedAt   int64   `json:"created_at"`
	CompletedAt float64 `json:"completed_at"`
	DurationMs  float64 `json:"duration_ms"`
}
