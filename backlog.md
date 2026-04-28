# Observer Feature Checklist

## Phase 1: Core Runtime Path (MVP)

### Configuration & Routing

- [x] Implement file-based host configuration loader (YAML)
- [x] Validate configuration at startup
- [x] Build in-memory routing table for host → upstream mapping
- [x] Handle invalid configuration with clear errors

### Proxy Layer

- [x] Implement basic HTTP reverse proxy
- [x] Route requests based on host configuration
- [x] Forward requests to upstream service
- [x] Return upstream responses transparently to client
- [x] Handle upstream failures (502 / 504 responses)

### Audit Job Model

- [x] Define request audit job structure
- [x] Define response audit job structure
- [x] Define failure audit job structure
- [x] Include metadata (host, path, method, status, request ID)
- [x] Add helper constructors: NewRequestJob, NewResponseJob, NewFailureJob

### Queue Layer

- [x] Implement in-memory queue for audit jobs
- [x] Implement enqueue operation
- [x] Support safe concurrent access
- [x] Handle graceful shutdown

### Proxy → Queue Integration

- [ ] Enqueue request audit job before forwarding upstream
- [ ] Enqueue response audit job after receiving response
- [ ] Enqueue failure audit job on upstream error
- [ ] Ensure queue operations do not block proxy path

### Worker Layer

- [ ] Implement worker pool
- [ ] Configure worker concurrency
- [ ] Continuously consume audit jobs from queue
- [ ] Handle graceful worker shutdown

### Persistence Layer (SQLite)

- [ ] Initialize SQLite database
- [ ] Design findings table schema
- [ ] Persist audit findings
- [ ] Persist request/response metadata
- [ ] Ensure data persists across restarts

### Minimal Auditing (Skeleton)

- [ ] Implement basic auditing pipeline (no contracts yet)
- [ ] Generate finding for upstream failures
- [ ] Generate finding for unknown hosts
- [ ] Generate basic “request observed” / “response observed” records

---

## Phase 2: Contract-Based Auditing

### API Contract Integration

- [ ] Load API contract files per host
- [ ] Validate contract structure at startup
- [ ] Map contracts to routes and methods

### Request-Side API Auditing

- [ ] Validate request path exists in contract
- [ ] Validate HTTP method is allowed
- [ ] Validate content type
- [ ] Validate request body presence/absence
- [ ] Validate JSON format for request body

### Response-Side API Auditing

- [ ] Validate response status codes
- [ ] Validate response content type
- [ ] Validate JSON format for response body
- [ ] Validate basic schema conformance

### Resource Contract Integration

- [ ] Load resource contract files per host
- [ ] Validate resource contract structure
- [ ] Map resources to API operations

### Request-Side Resource Auditing

- [ ] Validate allowed fields in request body
- [ ] Enforce write permissions
- [ ] Enforce mutability constraints for updates

### Response-Side Resource Auditing

- [ ] Validate allowed fields in response body
- [ ] Detect sensitive field exposure
- [ ] Enforce read permissions

---

## Phase 3: Operability & Resilience

### Findings Access & Tooling

- [ ] Implement CLI to list findings
- [ ] Filter findings by host, severity, or type
- [ ] View detailed finding information

### Queue & Performance Controls

- [ ] Add queue size limits or backpressure handling
- [ ] Detect and log queue buildup
- [ ] Tune worker concurrency

### Logging & Observability

- [ ] Implement structured logging
- [ ] Add request ID tracking across layers
- [ ] Log proxy lifecycle events
- [ ] Log worker processing events
- [ ] Log persistence failures

### Fault Tolerance Improvements

- [ ] Handle queue overload scenarios
- [ ] Ensure proxy continues if worker/persistence fails
- [ ] Retry or log failed persistence operations

### Security Hardening

- [ ] Validate all configuration inputs strictly
- [ ] Mask sensitive data in logs and findings
- [ ] Avoid storing secrets in plaintext configs
- [ ] Prepare for future auth/authz support

---

## Phase 4: Future Enhancements

### Scalability Improvements

- [ ] Replace in-memory queue with external queue (optional)
- [ ] Support distributed worker processing
- [ ] Improve database performance or migrate from SQLite

### Advanced Auditing

- [ ] Support complex multi-resource validation
- [ ] Support aggregation and derived field validation
- [ ] Add customizable audit rules

### Platform Evolution

- [ ] Add UI/dashboard for findings visualization
- [ ] Add metrics and monitoring integration
- [ ] Support multi-instance deployment
