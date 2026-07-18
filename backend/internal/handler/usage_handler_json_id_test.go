package handler

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBatchAPIKeysUsageRequestAcceptsStringAPIKeyIDs(t *testing.T) {
	var req BatchAPIKeysUsageRequest
	require.NoError(t, json.Unmarshal([]byte(`{"api_key_ids":["1192354569694740481",1002]}`), &req))
	require.Equal(t, []int64{1192354569694740481, 1002}, req.APIKeyIDs.Int64s())
}
