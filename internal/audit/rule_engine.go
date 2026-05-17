package audit

import (
	"fmt"
	"observer/internal/config"
	"os"
	"path/filepath"
	"strings"
)

type RuleID string

const (
	// RuleProxyUpstreamFailure applies when the proxy successfully matches a host
	// and route target, but cannot complete the upstream request due to a
	// non-timeout proxy error.
	//
	// Example:
	//   - upstream service is not running
	//   - connection is refused
	//   - upstream connection is reset
	//
	// This rule is evaluated against FailureJob values.
	RuleProxyUpstreamFailure RuleID = "proxy.upstream_failure"

	// RuleProxyUpstreamTimeout applies when the proxy successfully matches a host
	// and route target, but the upstream request exceeds the configured timeout.
	//
	// Example:
	//   - upstream does not accept the connection before Dial timeout
	//   - upstream accepts the connection but does not return response headers
	//     before ResponseHeaderTimeout
	//
	// This rule is evaluated against FailureJob values.
	RuleProxyUpstreamTimeout RuleID = "proxy.upstream_timeout"

	// RuleRequestPathDoesNotExist applies when an incoming request path cannot be
	// matched to any path defined in the OpenAPI contract for the selected host.
	//
	// Example:
	//   - request:  GET /admin
	//   - contract: /users, /users/{id}, /health
	//
	// This rule should run before method, content type, and body validation because
	// those checks require a matching contract path.
	//
	// This rule is evaluated against RequestJob values.
	RuleRequestPathDoesNotExist RuleID = "request.path_does_not_exist"

	// RuleRequestMethodNotAllowed applies when the request path exists in the
	// OpenAPI contract, but the specific HTTP method is not defined for that path.
	//
	// Example:
	//   - request:  DELETE /users
	//   - contract: GET /users and POST /users only
	//
	// This rule should run after path matching succeeds, but before request body
	// validation because the operation definition is needed for deeper checks.
	//
	// This rule is evaluated against RequestJob values.
	RuleRequestMethodNotAllowed RuleID = "request.method_not_allowed"

	// RuleRequestContentTypeNotAllowed applies when the request has a body and the
	// Content-Type header does not match any media type allowed by the OpenAPI
	// operation's requestBody.content map.
	//
	// Example:
	//   - request Content-Type: text/plain
	//   - contract allows: application/json
	//
	// This validates the declared media type only. It does not prove the body is
	// actually valid JSON, XML, multipart data, etc.
	//
	// This rule is evaluated against RequestJob values.
	RuleRequestContentTypeNotAllowed RuleID = "request.content_type_not_allowed"

	// RuleRequestBodyMissing applies when the OpenAPI operation declares a required
	// request body, but the captured request body is empty.
	//
	// Example:
	//   - contract: requestBody.required = true
	//   - request: POST /users with no body
	//
	// This rule should run after the path and method have been resolved to an
	// OpenAPI operation.
	//
	// This rule is evaluated against RequestJob values.
	RuleRequestBodyMissing RuleID = "request.body_missing"

	// RuleRequestBodyNotAllowed applies when the OpenAPI operation does not define
	// a requestBody, but the incoming request includes a body.
	//
	// Example:
	//   - contract: GET /health has no requestBody
	//   - request: GET /health with a JSON body
	//
	// This catches clients sending payloads to operations that are expected to be
	// bodyless.
	//
	// This rule is evaluated against RequestJob values.
	RuleRequestBodyNotAllowed RuleID = "request.body_not_allowed"

	// RuleRequestInvalidBodyFormat applies when the request body does not match the
	// expected non-JSON media format declared by the OpenAPI contract.
	//
	// Example future uses:
	//	 - application/json body cannot be parsed as JSON
	//   - multipart/form-data body cannot be parsed as multipart data
	//   - application/xml body cannot be parsed as XML
	//   - text/csv body cannot be parsed as CSV
	//
	// This is a generic extension point for media-type-specific validators beyond
	// JSON. It should run only after content type validation determines which media
	// type applies.
	//
	// This rule is evaluated against RequestJob values.
	RuleRequestInvalidBodyFormat RuleID = "request.invalid_body_format"

	// RuleRequestBodySchemaInvalid applies when the request body is syntactically
	// valid for its media type, but does not conform to the schema declared by the
	// OpenAPI contract.
	//
	// For the initial implementation, this rule should focus on JSON request bodies
	// only, such as application/json and application/*+json.
	//
	// Example:
	//   - contract: POST /users requires CreateUserRequest
	//   - schema requires: email, displayName
	//   - request body: {"email":"john@example.com"}
	//   - result: missing required field "displayName"
	//
	// This rule should run only after body format validation has succeeded. It
	// assumes the request body can already be parsed, then checks the parsed value
	// against a supported subset of the OpenAPI schema.
	//
	// This rule is evaluated against RequestJob values.
	RuleRequestBodySchemaInvalid RuleID = "request.body_schema_invalid"

	// RuleResponseStatusCodeNotDefined applies when the upstream response status
	// code is not defined in the OpenAPI operation's responses map.
	//
	// Example:
	//   - response status: 500
	//   - contract responses: 200, 400, 404
	//
	// This rule should run after the request path and method have been resolved to
	// an OpenAPI operation because response validation depends on the selected
	// operation definition.
	//
	// This rule is evaluated against ResponseJob values.
	RuleResponseStatusCodeNotDefined RuleID = "response.status_code_not_defined"

	// RuleResponseContentTypeNotAllowed applies when the upstream response includes
	// a body and the Content-Type header does not match any media type allowed by
	// the OpenAPI response definition for the returned status code.
	//
	// Example:
	//   - response Content-Type: text/plain
	//   - contract allows: application/json
	//
	// This validates the declared media type only. It does not prove the body is
	// actually valid JSON, XML, multipart data, etc.
	//
	// This rule is evaluated against ResponseJob values.
	RuleResponseContentTypeNotAllowed RuleID = "response.content_type_not_allowed"

	// RuleResponseBodyMissing applies when the OpenAPI response definition declares
	// response content for the returned status code, but the captured response body
	// is empty.
	//
	// Example:
	//   - contract: 200 response defines application/json content
	//   - response: 200 OK with no body
	//
	// This rule should run after the response status code has been matched to an
	// OpenAPI response definition.
	//
	// This rule is evaluated against ResponseJob values.
	RuleResponseBodyMissing RuleID = "response.body_missing"

	// RuleResponseBodyNotAllowed applies when the OpenAPI response definition does
	// not declare response content for the returned status code, but the upstream
	// response includes a body.
	//
	// Example:
	//   - contract: 204 response has no content
	//   - response: 204 No Content with a JSON body
	//
	// This catches upstream services returning payloads for responses that are
	// expected to be bodyless.
	//
	// This rule is evaluated against ResponseJob values.
	RuleResponseBodyNotAllowed RuleID = "response.body_not_allowed"

	// RuleResponseInvalidBodyFormat applies when the response body does not match
	// the expected non-JSON media format declared by the OpenAPI contract.
	//
	// Example future uses:
	//   - application/json body cannot be parsed as JSON
	//   - application/xml body cannot be parsed as XML
	//   - text/csv body cannot be parsed as CSV
	//
	// This is a generic extension point for media-type-specific validators beyond
	// JSON. It should run only after content type validation determines which media
	// type applies.
	//
	// This rule is evaluated against ResponseJob values.
	RuleResponseInvalidBodyFormat RuleID = "response.invalid_body_format"

	// RuleResponseBodySchemaInvalid applies when the response body is syntactically
	// valid for its media type, but does not conform to the schema declared by the
	// OpenAPI contract.
	//
	// For the initial implementation, this rule should focus on JSON response bodies
	// only, such as application/json and application/*+json.
	//
	// Example:
	//   - contract: GET /users/{id} returns User
	//   - schema requires: id, email, displayName
	//   - response body: {"id":"123","email":"john@example.com"}
	//   - result: missing required field "displayName"
	//
	// This rule should run only after body format validation has succeeded. It
	// assumes the response body can already be parsed, then checks the parsed value
	// against a supported subset of the OpenAPI schema.
	//
	// This rule is evaluated against ResponseJob values.
	RuleResponseBodySchemaInvalid RuleID = "response.body_schema_invalid"
)

type Rule interface {
	ID() RuleID
	Title() string
	AppliesTo() []JobType
	Check(ctx RuleContext, job Job, jobID string) ([]Finding, error)
}

type RuleContext struct {
	Contracts *ContractRegistry
}

type RuleEngine struct {
	rules    []Rule
	registry *ContractRegistry
}

func NewRuleEngine(registry *ContractRegistry) *RuleEngine {
	return &RuleEngine{
		rules:    getRules(),
		registry: registry,
	}
}

type ContractRegistry struct {
	contracts map[string]OpenAPIDoc
}

func (r *ContractRegistry) HasContract(host string) bool {
	if r == nil {
		return false
	}

	_, ok := r.contracts[strings.ToLower(host)]
	return ok
}

func (r *ContractRegistry) FindOperation(host, method, path string) (*OpenAPIOperation, bool) {
	doc, ok := r.contracts[strings.ToLower(host)]
	if !ok {
		return nil, false
	}

	return doc.FindOpenAPIOperation(method, path)
}

func (r *ContractRegistry) FindPathItem(host, path string) (*OpenAPIPathItem, bool) {
	doc, ok := r.contracts[strings.ToLower(host)]
	if !ok {
		return nil, false
	}

	if pathItem, ok := doc.Paths[path]; ok {
		return &pathItem, true
	}

	for contractPath, pathItem := range doc.Paths {
		if matchOpenAPIPath(contractPath, path) {
			return &pathItem, true
		}
	}

	return nil, false
}

func (r *ContractRegistry) FindMethod(host, method, path string) (*OpenAPIOperation, bool) {
	pathItem, ok := r.FindPathItem(host, path)
	if !ok {
		return nil, false
	}

	op := pathItem.OperationForMethod(method)
	if op == nil {
		return nil, false
	}

	return op, true
}

func (r *ContractRegistry) FindRequestContentType(host, method, path, contentType string) (mt OpenAPIMediaType, applies, found bool) {
	pathItem, ok := r.FindPathItem(host, path)
	if !ok {
		return OpenAPIMediaType{}, false, false
	}

	op := pathItem.OperationForMethod(method)
	if op == nil || op.RequestBody == nil {
		return OpenAPIMediaType{}, false, false
	}

	contentType = normalizeMediaType(contentType)

	mediaType, ok := op.RequestBody.Content[contentType]
	if !ok {
		return OpenAPIMediaType{}, true, false
	}

	return mediaType, true, true
}

func (r *ContractRegistry) FindResponseContentType(host, method, path, status, contentType string) (mt OpenAPIMediaType, applies, found bool) {
	pathItem, ok := r.FindPathItem(host, path)
	if !ok {
		return OpenAPIMediaType{}, false, false
	}

	op := pathItem.OperationForMethod(method)
	if op == nil {
		return OpenAPIMediaType{}, false, false
	}

	res, ok := op.Responses[status]
	if !ok {
		res, ok = op.Responses["default"]
		if !ok {
			return OpenAPIMediaType{}, false, false
		}
	}

	if len(res.Content) == 0 {
		return OpenAPIMediaType{}, false, false
	}

	contentType = normalizeMediaType(contentType)

	mediaType, ok := res.Content[contentType]
	if !ok {
		return OpenAPIMediaType{}, true, false
	}

	return mediaType, true, true
}

func normalizeMediaType(contentType string) string {
	contentType = strings.ToLower(strings.TrimSpace(contentType))

	if i := strings.Index(contentType, ";"); i >= 0 {
		contentType = strings.TrimSpace(contentType[:i])
	}

	return contentType
}

func (r *ContractRegistry) FindRequestBody(host, method, path string) (*OpenAPIRequestBody, bool) {
	op, ok := r.FindOperation(host, method, path)
	if !ok {
		return nil, false
	}

	return op.RequestBody, true
}

func (r *ContractRegistry) FindResponseBody(host, method, path, status string) (*OpenAPIResponse, bool) {
	op, ok := r.FindOperation(host, method, path)
	if !ok {
		return nil, false
	}

	res, ok := op.Responses[status]
	if !ok {
		res, ok = op.Responses["default"]
		if !ok {
			return nil, false
		}
	}

	return &res, true
}

func (r *ContractRegistry) ResolveSchemaRef(host, ref string) (*OpenAPISchema, bool) {
	doc, ok := r.contracts[strings.ToLower(host)]
	if !ok {
		return nil, false
	}

	return doc.ResolveSchemaRef(ref)
}

func NewContractRegistry(hosts map[string]config.Host) (*ContractRegistry, error) {
	contracts := make(map[string]OpenAPIDoc)

	baseDir, err := os.UserConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(baseDir, "observer", "config.yaml")

	for hostName, host := range hosts {
		if host.APIContractPath == "" {
			continue
		}

		contractPath := host.APIContractPath
		if !filepath.IsAbs(contractPath) {
			contractPath = filepath.Join(filepath.Dir(configPath), contractPath)
		}

		contract, err := LoadOpenAPIDocument(contractPath)
		if err != nil {
			return nil, fmt.Errorf("load api contract for host %q: %w", hostName, err)
		}

		contracts[strings.ToLower(hostName)] = contract
	}

	return &ContractRegistry{
		contracts: contracts,
	}, nil
}

func getRules() []Rule {
	return []Rule{
		UpstreamFailureRule{},
		UpstreamTimeoutRule{},
		RequestPathDoesNotExistRule{},
		RequestMethodNotAllowedRule{},
		RequestContentTypeNotAllowed{},
		RequestBodyMissing{},
		RequestBodyNotAllowed{},
		RequestBodyInvalidFormat{},
		RequestBodySchemaInvalid{},
		ResponseStatusCodeRule{},
		ResponseContentTypeNotAllowed{},
		ResponseBodyMissing{},
		ResponseBodyNotAllowed{},
		ResponseBodyInvalidFormat{},
		ResponseBodySchemaInvalid{},
	}
}

// TODO: Update this to allow for custom rules to still apply even if there is no OpenAPI contract (ex. Rule for checking request query for possible SQL injection)
func (e *RuleEngine) Evaluate(job Job, jobID string) ([]Finding, error) {
	if e == nil || e.registry == nil {
		return nil, nil
	}

	meta := job.Metadata()

	if !e.registry.HasContract(meta.Host) {
		return nil, nil
	}

	var findings []Finding

	ctx := RuleContext{
		Contracts: e.registry,
	}

	for _, rule := range e.rules {
		if !ruleApplies(rule, job.JobType()) {
			continue
		}

		ruleFindings, err := rule.Check(ctx, job, jobID)
		if err != nil {
			return nil, err
		}

		findings = append(findings, ruleFindings...)
	}

	return findings, nil
}

func ruleApplies(rule Rule, jobType JobType) bool {
	for _, supported := range rule.AppliesTo() {
		if supported == jobType {
			return true
		}
	}

	return false
}
