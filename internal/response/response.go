// Package response contains the public JSON contract shared by every API endpoint.
package response

import (
	"net/http"
	"strings"
)

const APIVersion = "v1"

// Envelope is the stable response shape exposed to API clients.
type Envelope struct {
	Success   bool          `json:"success"`
	Version   string        `json:"version"`
	Timestamp string        `json:"timestamp"`
	Message   string        `json:"message"`
	Data      interface{}   `json:"data"`
	Meta      interface{}   `json:"meta"`
	Error     *Error        `json:"error"`
	RequestID string        `json:"requestId"`
}

// Error describes a machine-readable API failure.
type Error struct {
	Code    string      `json:"code"`
	Details interface{} `json:"details"`
}

// Success builds a successful API response.
func Success(message string, data, meta interface{}, requestID, timestamp string) Envelope {
	return Envelope{
		Success:   true,
		Version:   APIVersion,
		Timestamp: timestamp,
		Message:   message,
		Data:      data,
		Meta:      meta,
		Error:     nil,
		RequestID: requestID,
	}
}

// Failure builds a failed API response. Error responses never expose data.
func Failure(status int, message string, details interface{}, requestID, timestamp string) Envelope {
	return Envelope{
		Success:   false,
		Version:   APIVersion,
		Timestamp: timestamp,
		Message:   message,
		Data:      nil,
		Meta:      nil,
		Error: &Error{
			Code:    CodeForStatus(status, message),
			Details: details,
		},
		RequestID: requestID,
	}
}

// CodeForStatus maps HTTP failures to the documented, stable error identifiers.
func CodeForStatus(status int, message string) string {
	if status == http.StatusBadRequest && strings.Contains(strings.ToLower(message), "validation") {
		return "VALIDATION_ERROR"
	}
	if status == http.StatusBadRequest && strings.Contains(strings.ToLower(message), "insufficient stock") {
		return "INSUFFICIENT_STOCK"
	}
	switch status {
	case http.StatusBadRequest:
		return "INVALID_REQUEST"
	case http.StatusUnauthorized:
		return "UNAUTHORIZED"
	case http.StatusForbidden:
		return "FORBIDDEN"
	case http.StatusNotFound:
		return "RESOURCE_NOT_FOUND"
	case http.StatusConflict:
		return "CONFLICT"
	case http.StatusServiceUnavailable:
		return "SERVICE_UNAVAILABLE"
	default:
		return "INTERNAL_SERVER_ERROR"
	}
}
