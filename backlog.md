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

- [x] Enqueue request audit job before forwarding upstream
- [x] Enqueue response audit job after receiving response
- [x] Enqueue failure audit job on upstream error
- [x] Ensure queue operations do not block proxy path

### Worker Layer

- [x] Implement worker pool
- [x] Configure worker concurrency
- [x] Continuously consume audit jobs from queue
- [x] Handle graceful worker shutdown

### Persistence Layer (SQLite)

- [x] Initialize SQLite database
- [x] Design findings table schema
- [x] Persist audit findings
- [x] Persist request/response metadata
- [x] Ensure data persists across restarts

### Minimal Auditing (Skeleton)

- [x] Implement basic auditing pipeline
- [x] Generate finding for upstream failures
- [x] Generate finding for upstream timeouts
- [x] Persist findings linked to audit jobs

---

## Phase 2: API Contract Auditing

### API Contract Integration

- [x] Load API contract files per host
- [x] Validate contract structure at startup
- [x] Map contracts to routes and methods
- [x] Support exact path matching
- [x] Support basic OpenAPI path parameter matching

### Request-Side API Auditing

- [x] Validate request path exists in contract
- [x] Validate HTTP method is allowed
- [x] Validate request content type
- [x] Validate request body presence/absence
- [x] Validate request body format based on contract media type
- [x] Validate basic schema conformance for JSON request bodies

### Response-Side API Auditing

- [x] Validate response status codes
- [x] Validate response content type
- [x] Validate response body presence/absence
- [x] Validate response body format based on contract media type
- [x] Validate basic schema conformance for JSON response bodies

---

## Phase 3: Custom Rules Contract Auditing

### Custom Rules Contract Integration

- [x] Load custom rules contract files per host
- [x] Validate custom rules contract structure at startup
- [x] Bind custom rules contracts to configured hosts
- [x] Support custom rules contracts in YAML
- [x] Support custom rules contracts in JSON
- [x] Generate clear errors for invalid custom rules contracts
- [x] Precompile custom rule regex patterns at startup

### Custom Rule Engine

- [x] Scope custom rules by configured host
- [x] Skip disabled custom rules
- [x] Match custom rules by job type
- [x] Match custom rules by HTTP method
- [x] Match custom rules by exact path
- [x] Match custom rules by OpenAPI-style path parameters
- [x] Match custom rules by wildcard path patterns
- [x] Dispatch custom rules by rule type
- [x] Evaluate request custom rules
- [x] Evaluate response custom rules

### Custom Rule Type Evaluation

- [x] Evaluate path rules using regex patterns
- [x] Evaluate query rules using exact query parameter matches
- [x] Evaluate query rules using regex patterns
- [x] Evaluate header rules using exact header matches
- [x] Evaluate header rules using regex patterns
- [x] Evaluate body field rules using exact field path matches
- [x] Evaluate body field rules using regex patterns
- [x] Support dot-path matching for nested JSON body fields
- [x] Support dot-path regex matching for nested JSON body values

### Custom Rule Findings

- [x] Generate findings for custom rule violations
- [x] Include custom rule ID in generated findings
- [x] Include custom rule title in generated findings
- [x] Include custom rule message in generated findings
- [x] Store custom rule findings using the same persistence flow as built-in findings

---

## Phase 4: Operability & Resilience

### Findings Access & Tooling

- [ ] Implement CLI to list findings
- [ ] Filter findings by host, severity, rule ID, or type
- [ ] View detailed finding information
- [ ] List audit jobs and request history
- [ ] Inspect findings linked to a specific request ID

### Queue & Performance Controls

- [ ] Add queue size limits or backpressure handling
- [ ] Detect and log queue buildup
- [ ] Tune worker concurrency
- [ ] Track dropped audit jobs

### Logging & Observability

- [ ] Implement structured logging
- [ ] Add request ID tracking across layers
- [ ] Log proxy lifecycle events
- [ ] Log worker processing events
- [ ] Log audit rule execution failures
- [ ] Log persistence failures
- [ ] Add basic metrics for queue depth, worker throughput, and finding count

### Fault Tolerance Improvements

- [ ] Handle queue overload scenarios
- [ ] Ensure proxy continues if worker/persistence fails
- [ ] Retry or log failed persistence operations
- [ ] Improve graceful shutdown behavior for in-flight audit jobs

### Security Hardening

- [ ] Validate all configuration inputs strictly
- [ ] Validate API contract inputs strictly
- [ ] Validate custom rules contract inputs strictly
- [ ] Mask sensitive data in logs and findings
- [ ] Avoid storing secrets in plaintext configs
- [ ] Track direct remote address in addition to forwarded headers
- [ ] Prepare for future auth/authz support

---

## Phase 5: Future Enhancements

### Scalability Improvements

- [ ] Replace in-memory queue with external queue or message broker
- [ ] Support distributed worker processing
- [ ] Improve database performance or migrate from SQLite
- [ ] Add retention policies for audit jobs and findings
- [ ] Add configurable audit sampling or rule-specific enablement

### Advanced API Contract Auditing

- [ ] Support advanced JSON Schema/OpenAPI validation features
- [ ] Support `oneOf`
- [ ] Support `anyOf`
- [ ] Support `allOf`
- [ ] Support `enum`
- [ ] Support `nullable`
- [ ] Support `additionalProperties`
- [ ] Support string and numeric constraints
- [ ] Support advanced `format` validation

### Advanced Custom Rules

- [ ] Support authorization expectation rules
- [ ] Support route-specific security policies
- [ ] Support compliance-oriented checks
- [ ] Support environment-specific rule overrides
- [ ] Support reusable rule templates
- [ ] Support custom rule groups or profiles

### Platform Evolution

- [ ] Add UI/dashboard for findings visualization
- [ ] Add metrics and monitoring integration
- [ ] Support multi-instance deployment
- [ ] Add exportable audit reports
- [ ] Add CI/CD integration for contract and rules validation
