package reseller

// Snapshot freezes ownership and prices for one admitted request or async task.
type Snapshot struct {
	UserID     int64   `json:"user_id"`
	OwnerID    int64   `json:"owner_id"`
	GroupID    int64   `json:"group_id"`
	Multiplier float64 `json:"multiplier"`
	TextRate   float64 `json:"text_rate"`
	ImageRate  float64 `json:"image_rate"`
	VideoRate  float64 `json:"video_rate"`
}
