package audit

import (
	"bytes"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net"
	"net/http"
	"time"
)

type JobType string

const (
	RequestJobType  JobType = "request"
	ResponseJobType JobType = "response"
	FailureJobType  JobType = "failure"
)

type Metadata struct {
	RequestID  string
	Host       string
	Method     string
	Path       string
	Query      string
	Upstream   string
	Status     int
	Timestamp  time.Time
	DurationMs int64
}

type RequestJob struct {
	Type    JobType
	Meta    Metadata
	Headers http.Header
	Body    []byte
}

type ResponseJob struct {
	Type    JobType
	Meta    Metadata
	Headers http.Header
	Body    []byte
}

type FailureJob struct {
	Type  JobType
	Meta  Metadata
	Error string
}

// NOTE:
// We read the full request/response body here to capture it for auditing,
// then restore the body so the proxy/client can continue using it.
// This is synchronous and may add latency and memory overhead for large payloads.
// Future work:
//   - add a configurable max body size
//   - move body capture to the worker layer
//   - or use a streaming (io.TeeReader) approach to avoid full buffering
func NewRequestJob(r *http.Request, upstream string, start time.Time) *RequestJob {
	requestID := getOrCreateRequestID(r)

	host := r.Header.Get("X-Original-Host")
	if host == "" {
		host = r.Host
	}

	var body []byte
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err == nil {
			body = b
		}

		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	return &RequestJob{
		Type: RequestJobType,
		Meta: Metadata{
			RequestID: requestID,
			Host:      host,
			Method:    r.Method,
			Path:      r.URL.Path,
			Query:     r.URL.RawQuery,
			Upstream:  upstream,
			Timestamp: start,
		},
		Headers: r.Header.Clone(),
		Body:    body,
	}
}

func NewResponseJob(r *http.Response, upstream string) *ResponseJob {
	requestID := getOrCreateRequestID(r.Request)

	var duration int64
	start, err := time.Parse(time.RFC3339Nano, r.Request.Header.Get("X-Request-Timestamp"))
	if err == nil {
		duration = time.Since(start).Milliseconds()
	}

	host := r.Request.Header.Get("X-Original-Host")
	if host == "" {
		host = r.Request.Host
	}

	var body []byte
	if r.Body != nil {
		b, err := io.ReadAll(r.Body)
		if err == nil {
			body = b
		}

		_ = r.Body.Close()
		r.Body = io.NopCloser(bytes.NewReader(body))
	}

	return &ResponseJob{
		Type: ResponseJobType,
		Meta: Metadata{
			RequestID:  requestID,
			Host:       host,
			Method:     r.Request.Method,
			Path:       r.Request.URL.Path,
			Query:      r.Request.URL.RawQuery,
			Upstream:   upstream,
			Status:     r.StatusCode,
			Timestamp:  time.Now().UTC(),
			DurationMs: duration,
		},
		Headers: r.Header.Clone(),
		Body:    body,
	}
}

func NewFailureJob(r *http.Request, upstream string, err error) *FailureJob {
	requestID := getOrCreateRequestID(r)

	status := http.StatusBadGateway
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		status = http.StatusGatewayTimeout
	}

	start, _ := time.Parse(time.RFC3339Nano, r.Header.Get("X-Request-Timestamp"))

	var duration int64
	start, parseErr := time.Parse(time.RFC3339Nano, r.Header.Get("X-Request-Timestamp"))
	if parseErr == nil {
		duration = time.Since(start).Milliseconds()
	}

	host := r.Header.Get("X-Original-Host")
	if host == "" {
		host = r.Host
	}

	return &FailureJob{
		Type: FailureJobType,
		Meta: Metadata{
			RequestID:  requestID,
			Host:       host,
			Method:     r.Method,
			Path:       r.URL.Path,
			Query:      r.URL.RawQuery,
			Upstream:   upstream,
			Status:     status,
			DurationMs: duration,
			Timestamp:  time.Now().UTC(),
		},
		Error: err.Error(),
	}
}

func getOrCreateRequestID(r *http.Request) string {
	if r == nil || r.Header == nil {
		return newRequestID()
	}

	if id := r.Header.Get("X-Request-ID"); id != "" {
		return id
	}

	return newRequestID()
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}

	return time.Now().UTC().Format("20060102150405.000000000")
}
