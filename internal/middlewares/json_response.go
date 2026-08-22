package middlewares

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"posku/internal/response"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

const requestIDContextKey = "request_id"

// JSONResponseMiddleware converts legacy controller output into the single API
// response contract at the HTTP boundary.
func JSONResponseMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := newRequestID()
		c.Set(requestIDContextKey, requestID)
		c.Header("X-Request-ID", requestID)

		writer := &bufferedResponseWriter{ResponseWriter: c.Writer, status: http.StatusOK}
		c.Writer = writer
		c.Next()

		// CORS preflight deliberately has no response body.
		if writer.status == http.StatusNoContent || writer.buffer.Len() == 0 {
			return
		}

		payload := decodePayload(writer.buffer.Bytes())
		timestamp := time.Now().UTC().Format(time.RFC3339)
		var envelope response.Envelope
		if writer.status >= http.StatusBadRequest {
			message, details := errorPayload(payload, http.StatusText(writer.status))
			envelope = response.Failure(writer.status, message, details, requestID, timestamp)
		} else {
			message, data := successPayload(payload)
			envelope = response.Success(message, data, nil, requestID, timestamp)
		}

		body, err := json.Marshal(envelope)
		if err != nil {
			body, _ = json.Marshal(response.Failure(http.StatusInternalServerError, "Failed to encode response", nil, requestID, timestamp))
			writer.status = http.StatusInternalServerError
		}

		writer.ResponseWriter.Header().Set("Content-Type", "application/json; charset=utf-8")
		writer.ResponseWriter.WriteHeader(writer.status)
		_, _ = writer.ResponseWriter.Write(body)
	}
}

type bufferedResponseWriter struct {
	gin.ResponseWriter
	buffer        bytes.Buffer
	status        int
	headerWritten bool
}

func (w *bufferedResponseWriter) WriteHeader(status int) {
	if !w.headerWritten {
		w.status = status
	}
}

func (w *bufferedResponseWriter) WriteHeaderNow() { w.headerWritten = true }

func (w *bufferedResponseWriter) Write(data []byte) (int, error) {
	return w.buffer.Write(data)
}

func (w *bufferedResponseWriter) WriteString(value string) (int, error) {
	return w.buffer.WriteString(value)
}

func (w *bufferedResponseWriter) Status() int { return w.status }

func (w *bufferedResponseWriter) Size() int { return w.buffer.Len() }

func (w *bufferedResponseWriter) Written() bool { return w.headerWritten }

func decodePayload(body []byte) interface{} {
	var payload interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil
	}
	return payload
}

func successPayload(payload interface{}) (string, interface{}) {
	if object, ok := payload.(map[string]interface{}); ok && len(object) == 1 {
		if message, ok := object["message"].(string); ok {
			return message, nil
		}
	}
	return "Request successfully processed", payload
}

func errorPayload(payload interface{}, fallback string) (string, interface{}) {
	if object, ok := payload.(map[string]interface{}); ok {
		if message, ok := object["error"].(string); ok && message != "" {
			return message, nil
		}
		if message, ok := object["message"].(string); ok && message != "" {
			return message, nil
		}
	}
	return fallback, nil
}

func newRequestID() string {
	bytes := make([]byte, 12)
	if _, err := rand.Read(bytes); err == nil {
		return "req_" + hex.EncodeToString(bytes)
	}
	// The random source is expected to be available; time still keeps the ID useful for tracing if it is not.
	return "req_" + strings.ReplaceAll(time.Now().UTC().Format("20060102150405.000000000"), ".", "")
}
