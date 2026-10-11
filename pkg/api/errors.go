package api

import (
	"encoding/json"
	"fmt"
	"strings"
)

func parseAPIError(status int, body []byte) error {
	msg := extractAPIErrorMessage(body)
	if msg == "" {
		msg = strings.TrimSpace(string(body))
	}
	if msg == "" {
		msg = "request failed"
	}
	return fmt.Errorf("API error (%d): %s", status, msg)
}

func extractAPIErrorMessage(body []byte) string {
	var errorResp ErrorResponse
	if err := json.Unmarshal(body, &errorResp); err == nil {
		if errorResp.Detail != "" {
			return errorResp.Detail
		}
		if errorResp.Message != "" {
			return errorResp.Message
		}
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(body, &obj); err != nil {
		return ""
	}
	if msg := stringFromRaw(obj["detail"]); msg != "" {
		return msg
	}
	if msg := stringFromRaw(obj["message"]); msg != "" {
		return msg
	}
	var nested map[string]json.RawMessage
	if raw, ok := obj["detail"]; ok && json.Unmarshal(raw, &nested) == nil {
		if msg := stringFromRaw(nested["message"]); msg != "" {
			return msg
		}
		if msg := stringFromRaw(nested["detail"]); msg != "" {
			return msg
		}
	}
	return ""
}

func stringFromRaw(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return strings.TrimSpace(s)
	}
	return ""
}
