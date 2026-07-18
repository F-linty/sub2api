package admin

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

type jsonInt64Slice []int64

type jsonInt64 int64

func (v *jsonInt64) UnmarshalJSON(data []byte) error {
	var n int64
	if err := json.Unmarshal(data, &n); err == nil {
		*v = jsonInt64(n)
		return nil
	}
	var text string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	text = strings.TrimSpace(text)
	if text == "" {
		return fmt.Errorf("empty int64 id")
	}
	n, err := strconv.ParseInt(text, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid int64 id %q: %w", text, err)
	}
	*v = jsonInt64(n)
	return nil
}

func (v jsonInt64) Int64() int64 {
	return int64(v)
}

func jsonInt64Ptr(v *jsonInt64) *int64 {
	if v == nil {
		return nil
	}
	n := v.Int64()
	return &n
}

func (s *jsonInt64Slice) UnmarshalJSON(data []byte) error {
	var raw []json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	out := make([]int64, 0, len(raw))
	for _, item := range raw {
		var n int64
		if err := json.Unmarshal(item, &n); err == nil {
			out = append(out, n)
			continue
		}
		var text string
		if err := json.Unmarshal(item, &text); err != nil {
			return err
		}
		text = strings.TrimSpace(text)
		if text == "" {
			return fmt.Errorf("empty int64 id")
		}
		n, err := strconv.ParseInt(text, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid int64 id %q: %w", text, err)
		}
		out = append(out, n)
	}
	*s = out
	return nil
}

func (s jsonInt64Slice) Int64s() []int64 {
	return []int64(s)
}

func jsonInt64SlicePtr(s *jsonInt64Slice) *[]int64 {
	if s == nil {
		return nil
	}
	ids := s.Int64s()
	return &ids
}

type jsonInt64SliceMap map[string]jsonInt64Slice

func (m jsonInt64SliceMap) Int64s() map[string][]int64 {
	if m == nil {
		return nil
	}
	out := make(map[string][]int64, len(m))
	for key, ids := range m {
		out[key] = ids.Int64s()
	}
	return out
}
