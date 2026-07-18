package admin

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestJSONInt64AcceptsNumberAndString(t *testing.T) {
	for _, raw := range []string{`1192354569694740481`, `"1192354569694740481"`} {
		var id jsonInt64
		require.NoError(t, json.Unmarshal([]byte(raw), &id))
		require.Equal(t, int64(1192354569694740481), id.Int64())
	}
}

func TestJSONInt64SliceAcceptsNumbersAndStrings(t *testing.T) {
	var ids jsonInt64Slice
	require.NoError(t, json.Unmarshal([]byte(`[1001,"1192354569694740481"]`), &ids))
	require.Equal(t, []int64{1001, 1192354569694740481}, ids.Int64s())
}

func TestUpdateAccountRequestAcceptsStringProxyID(t *testing.T) {
	var req UpdateAccountRequest
	require.NoError(t, json.Unmarshal([]byte(`{"proxy_id":"1192354569694740481"}`), &req))
	require.NotNil(t, req.ProxyID)
	require.Equal(t, int64(1192354569694740481), req.ProxyID.Int64())
}

func TestUpdateAccountRequestAcceptsStringGroupIDs(t *testing.T) {
	var req UpdateAccountRequest
	require.NoError(t, json.Unmarshal([]byte(`{"group_ids":["1192354569694740481",1002]}`), &req))
	require.NotNil(t, req.GroupIDs)
	require.Equal(t, []int64{1192354569694740481, 1002}, req.GroupIDs.Int64s())
}

func TestBatchGetUserAttributesRequestAcceptsStringUserIDs(t *testing.T) {
	var req BatchGetUserAttributesRequest
	require.NoError(t, json.Unmarshal([]byte(`{"user_ids":["1192354569694740481",1002]}`), &req))
	require.Equal(t, []int64{1192354569694740481, 1002}, req.UserIDs.Int64s())
}

func TestBatchUsersUsageRequestAcceptsStringUserIDs(t *testing.T) {
	var req BatchUsersUsageRequest
	require.NoError(t, json.Unmarshal([]byte(`{"user_ids":["1192354569694740481",1002]}`), &req))
	require.Equal(t, []int64{1192354569694740481, 1002}, req.UserIDs.Int64s())
}

func TestBatchAPIKeysUsageRequestAcceptsStringAPIKeyIDs(t *testing.T) {
	var req BatchAPIKeysUsageRequest
	require.NoError(t, json.Unmarshal([]byte(`{"api_key_ids":["1192354569694740481",1002]}`), &req))
	require.Equal(t, []int64{1192354569694740481, 1002}, req.APIKeyIDs.Int64s())
}
