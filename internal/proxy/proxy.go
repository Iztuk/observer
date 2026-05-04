package proxy

import (
	"cf-observer/internal/audit"
	"cf-observer/internal/config"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

type ProxyManager struct {
	Hosts  map[string]*ProxyTarget
	Logger *log.Logger
}

type ProxyTarget struct {
	Upstream *url.URL
	Proxy    *httputil.ReverseProxy

	Logger *log.Logger
}

type Observation struct {
	Timestamp time.Time `json:"timestamp"`
	Event     string    `json:"event"`

	RequestID string `json:"request_id"`

	Host     string `json:"host"`
	Method   string `json:"method"`
	Path     string `json:"path"`
	Query    string `json:"query"`
	Upstream string `json:"upstream"`

	Status     int   `json:"status,omitempty"`
	DurationMs int64 `json:"duration_ms,omitempty"`

	Error string `json:"error,omitempty"`

	RequestHeaders  map[string][]string `json:"request_headers,omitempty"`
	ResponseHeaders map[string][]string `json:"response_headers,omitempty"`
}

// TODO: Move the observation logging to the audit layer
func NewProxyManager(hosts map[string]config.Host, queue *audit.Queue, logger *log.Logger) (*ProxyManager, error) {
	pm := &ProxyManager{
		Hosts:  make(map[string]*ProxyTarget),
		Logger: logger,
	}

	for key, host := range hosts {
		h := host

		if host.Upstream == nil {
			return nil, fmt.Errorf("host %q has nil upstream", key)
		}

		// TODO: Redact sensitive request/response headers before logging
		rp := &httputil.ReverseProxy{
			Rewrite: func(pr *httputil.ProxyRequest) {
				pr.SetURL(h.Upstream)
				pr.SetXForwarded()

				originalHost := pr.In.Host
				pr.Out.Header.Set("X-Original-Host", originalHost)

				start := time.Now().UTC()
				pr.Out.Header.Set("X-Request-Timestamp", start.Format(time.RFC3339Nano))

				getOrCreateProxyRequestID(pr)

				job := audit.NewRequestJob(pr.Out, h.Upstream.String(), start)

				if pr.Out.Body != nil && pr.Out.Body != http.NoBody {
					pr.Out.Body = audit.NewCapturingBody(pr.Out.Body, func(b []byte) {
						job.Body = b

						if !queue.TryEnqueue(job) {
							logger.Printf("audit queue full; dropping request job")
							return
						}

					})
				} else {
					if !queue.TryEnqueue(job) {
						logger.Printf("audit queue full; dropping request job")
						return
					}
				}

			},
			ModifyResponse: func(r *http.Response) error {
				job := audit.NewResponseJob(r, h.Upstream.String())

				if r.Body != nil && r.Body != http.NoBody {
					r.Body = audit.NewCapturingBody(r.Body, func(b []byte) {
						job.Body = b

						if !queue.TryEnqueue(job) {
							logger.Printf("audit queue full; dropping response job")
							return
						}

					})
				} else {
					if !queue.TryEnqueue(job) {
						logger.Printf("audit queue full; dropping response job")
						return nil
					}
				}

				return nil
			},
			Transport: &http.Transport{
				ResponseHeaderTimeout: 5 * time.Second,
				DialContext: (&net.Dialer{
					Timeout: 5 * time.Second,
				}).DialContext,
			},
			ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
				job := audit.NewFailureJob(r, h.Upstream.String(), err)

				if !queue.TryEnqueue(job) {
					logger.Printf("audit queue full; dropping failure job")
				}

				http.Error(w, http.StatusText(job.Meta.Status), job.Meta.Status)
			},
		}

		pm.Hosts[strings.ToLower(key)] = &ProxyTarget{
			Upstream: host.Upstream,
			Proxy:    rp,
			Logger:   logger,
		}
	}

	return pm, nil
}

func (pm *ProxyManager) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host := normalizeHost(r.Host)

	target, ok := pm.Hosts[host]

	if !ok {
		pm.Logger.Printf("no route found for host=%s rawHost=%s", host, r.Host)
		http.NotFound(w, r)
		return
	}

	pm.Logger.Printf("routing host=%s to upstream=%s", host, target.Upstream.String())
	target.Proxy.ServeHTTP(w, r)
}

func normalizeHost(host string) string {
	if strings.Contains(host, ":") {
		h, _, err := net.SplitHostPort(host)
		if err == nil {
			return strings.ToLower(h)
		}
	}
	return strings.ToLower(host)
}

func getOrCreateProxyRequestID(pr *httputil.ProxyRequest) string {
	id := pr.In.Header.Get("X-Request-ID")
	if id == "" {
		id = newRequestID()
	}

	pr.Out.Header.Set("X-Request-ID", id)

	return id
}

func getOrCreateRequestID(r *http.Request) string {
	if r.Header == nil {
		r.Header = make(http.Header)
	}

	id := r.Header.Get("X-Request-ID")
	if id == "" {
		id = newRequestID()
		r.Header.Set("X-Request-ID", id)
	}

	return id
}

func newRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return hex.EncodeToString(b[:])
	}

	return time.Now().UTC().Format("20060102150405.000000000")
}

func writeObservation(logger *log.Logger, obs *Observation) {
	b, err := json.Marshal(obs)
	if err != nil {
		logger.Printf(`{"message":"failed to marshal observation","error":%q}`, err.Error())
		return
	}
	logger.Print(string(b))
}
