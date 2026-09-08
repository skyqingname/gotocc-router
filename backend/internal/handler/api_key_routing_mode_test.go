package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAPIKeyRoutingModeRejectsInvalidJSONValues(t *testing.T) {
	for _, payload := range []string{
		`{"routing_mode":null}`,
		`{"routing_mode":""}`,
		`{"routing_mode":"unknown"}`,
		`{"routing_mode":"auto","group_id":42}`,
	} {
		t.Run(payload, func(t *testing.T) {
			var create CreateAPIKeyRequest
			require.NoError(t, json.Unmarshal([]byte(payload), &create))
			require.Error(t, validateAPIKeyCreateRequest(create))
			var update UpdateAPIKeyRequest
			require.NoError(t, json.Unmarshal([]byte(payload), &update))
			require.Error(t, validateAPIKeyUpdateRequest(update))
		})
	}
}
