package apicompat

import "encoding/json"

// Token counts are also used as integers by the protocol converters. Track
// their wire presence separately so an explicit zero survives conversion
// without inventing counts that the upstream did not report.
type tokenDetailPresence map[string]json.RawMessage

func (fields tokenDetailPresence) count(name string, value int) *int {
	if value != 0 || (fields[name] != nil && string(fields[name]) != "null") {
		return &value
	}
	return nil
}

type tokenDetailsJSON struct {
	CachedTokens             *int `json:"cached_tokens,omitempty"`
	AudioTokens              *int `json:"audio_tokens,omitempty"`
	CacheCreationTokens      *int `json:"cache_creation_tokens,omitempty"`
	CacheWriteTokens         *int `json:"cache_write_tokens,omitempty"`
	ReasoningTokens          *int `json:"reasoning_tokens,omitempty"`
	AcceptedPredictionTokens *int `json:"accepted_prediction_tokens,omitempty"`
	RejectedPredictionTokens *int `json:"rejected_prediction_tokens,omitempty"`
}

func decodeTokenDetails(data []byte, target any) (tokenDetailPresence, error) {
	if err := json.Unmarshal(data, target); err != nil {
		return nil, err
	}
	var fields tokenDetailPresence
	err := json.Unmarshal(data, &fields)
	return fields, err
}

func (d *ResponsesInputTokensDetails) UnmarshalJSON(data []byte) error {
	type plain ResponsesInputTokensDetails
	var value plain
	fields, err := decodeTokenDetails(data, &value)
	if err != nil {
		return err
	}
	*d = ResponsesInputTokensDetails(value)
	d.presentFields = fields
	return nil
}

func (d ResponsesInputTokensDetails) MarshalJSON() ([]byte, error) {
	return json.Marshal(tokenDetailsJSON{
		CachedTokens:        d.presentFields.count("cached_tokens", d.CachedTokens),
		AudioTokens:         d.presentFields.count("audio_tokens", d.AudioTokens),
		CacheCreationTokens: d.presentFields.count("cache_creation_tokens", d.CacheCreationTokens),
		CacheWriteTokens:    d.presentFields.count("cache_write_tokens", d.CacheWriteTokens),
	})
}

func (d *ResponsesOutputTokensDetails) UnmarshalJSON(data []byte) error {
	type plain ResponsesOutputTokensDetails
	var value plain
	fields, err := decodeTokenDetails(data, &value)
	if err != nil {
		return err
	}
	*d = ResponsesOutputTokensDetails(value)
	d.presentFields = fields
	return nil
}

func (d ResponsesOutputTokensDetails) MarshalJSON() ([]byte, error) {
	return json.Marshal(tokenDetailsJSON{
		ReasoningTokens:          d.presentFields.count("reasoning_tokens", d.ReasoningTokens),
		AudioTokens:              d.presentFields.count("audio_tokens", d.AudioTokens),
		AcceptedPredictionTokens: d.presentFields.count("accepted_prediction_tokens", d.AcceptedPredictionTokens),
		RejectedPredictionTokens: d.presentFields.count("rejected_prediction_tokens", d.RejectedPredictionTokens),
	})
}

func (d *ChatTokenDetails) UnmarshalJSON(data []byte) error {
	type plain ChatTokenDetails
	var value plain
	fields, err := decodeTokenDetails(data, &value)
	if err != nil {
		return err
	}
	*d = ChatTokenDetails(value)
	d.presentFields = fields
	return nil
}

func (d ChatTokenDetails) MarshalJSON() ([]byte, error) {
	return json.Marshal(tokenDetailsJSON{
		CachedTokens:             d.presentFields.count("cached_tokens", d.CachedTokens),
		AudioTokens:              d.presentFields.count("audio_tokens", d.AudioTokens),
		CacheCreationTokens:      d.presentFields.count("cache_creation_tokens", d.CacheCreationTokens),
		CacheWriteTokens:         d.presentFields.count("cache_write_tokens", d.CacheWriteTokens),
		ReasoningTokens:          d.presentFields.count("reasoning_tokens", d.ReasoningTokens),
		AcceptedPredictionTokens: d.presentFields.count("accepted_prediction_tokens", d.AcceptedPredictionTokens),
		RejectedPredictionTokens: d.presentFields.count("rejected_prediction_tokens", d.RejectedPredictionTokens),
	})
}
