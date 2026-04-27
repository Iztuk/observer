package audit

import (
	"crypto/rand"
	"encoding/hex"
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

func NewRequestJob(r *http.Request, upstream string, start time.Time) *RequestJob {
	requestId := getOrCreateRequestID(r)

	host := r.Header.Get("X-Original-Host")
	if host == "" {
		host = r.Host
	}

	return &RequestJob{
		Type: RequestJobType,
		Meta: Metadata{
			RequestID: requestId,
			Host:      host,
			Method:    r.Method,
			Path:      r.URL.Path,
			Query:     r.URL.RawQuery,
			Upstream:  upstream,
			Timestamp: start,
		},
		Headers: r.Header.Clone(),
	}
}

func NewResponseJob(r *http.Response, upstream string) *ResponseJob {
	requestId := getOrCreateRequestID(r.Request)

	start, _ := time.Parse(time.RFC3339Nano, r.Request.Header.Get("X-Request-Timestamp"))

	var duration int64
	if !start.IsZero() {
		duration = time.Since(start).Milliseconds()
	}

	host := r.Request.Header.Get("X-Original-Host")
	if host == "" {
		host = r.Request.Host
	}

	return &ResponseJob{
		Type: ResponseJobType,
		Meta: Metadata{
			RequestID:  requestId,
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
	}
}

func NewFailureJob(r *http.Request, upstream string, err error) *FailureJob {
	requestId := getOrCreateRequestID(r)

	status := http.StatusBadGateway
	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		status = http.StatusGatewayTimeout
	}

	start, _ := time.Parse(time.RFC3339Nano, r.Header.Get("X-Request-Timestamp"))

	var duration int64
	if !start.IsZero() {
		duration = time.Since(start).Milliseconds()
	}

	host := r.Header.Get("X-Original-Host")
	if host == "" {
		host = r.Host
	}

	return &FailureJob{
		Type: FailureJobType,
		Meta: Metadata{
			RequestID:  requestId,
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
