package audit

import (
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

func (r HostRule) AppliesToJob(job Job) bool {
	if !r.Enabled {
		return false
	}

	if !jobTypeMatches(r.AppliesTo, job.JobType()) {
		return false
	}

	meta := job.Metadata()

	if !pathMatchesAny(r.Match.Paths, meta.Path) {
		return false
	}

	if !methodMatches(r.Match.Methods, meta.Method) {
		return false
	}

	var body any
	switch job.JobType() {
	case RequestJobType:
		req, ok := job.(*RequestJob)
		if !ok {
			return false
		}

		if !headerMatches(r.Match.Headers, req.Headers) {
			return false
		}

		if !bodyFieldMatches(r.Match.Fields, body) {
			return false
		}
	case ResponseJobType:
		res, ok := job.(*ResponseJob)
		if !ok {
			return false
		}

		if !headerMatches(r.Match.Headers, res.Headers) {
			return false
		}

		if !bodyFieldMatches(r.Match.Fields, body) {
			return false
		}
	default:
		return false
	}

	if !queryParamsMatches(r.Match.QueryParams, meta.Query) {
		return false
	}

	return true
}

func jobTypeMatches(supported []JobType, actual JobType) bool {
	if len(supported) == 0 {
		return true
	}

	for _, jobType := range supported {
		if jobType == actual {
			return true
		}
	}

	return false
}

func pathMatchesAny(patterns []string, actual string) bool {
	if len(patterns) == 0 {
		return true
	}

	for _, pattern := range patterns {
		if pathMatches(pattern, actual) {
			return true
		}
	}

	return false
}

func pathMatches(pattern, actual string) bool {
	if pattern == "*" {
		return true
	}

	if pattern == actual {
		return true
	}

	if strings.HasSuffix(pattern, "/*") {
		prefix := strings.TrimSuffix(pattern, "/*")
		return actual == prefix || strings.HasPrefix(actual, prefix+"/")
	}

	if matchOpenAPIPath(pattern, actual) {
		return true
	}

	return false
}

func methodMatches(methods []string, actual string) bool {
	if len(methods) == 0 {
		return true
	}

	for _, method := range methods {
		if strings.EqualFold(method, actual) {
			return true
		}
	}

	return false
}

func headerMatches(headers map[string][]string, actual http.Header) bool {
	if len(headers) == 0 {
		return true
	}

	if _, ok := headers["*"]; ok {
		return len(actual) > 0
	}

	for key := range headers {
		if actual.Get(key) != "" {
			return true
		}
	}

	return false
}

func queryParamsMatches(expected map[string][]string, rawQuery string) bool {
	if len(expected) == 0 {
		return true
	}

	vals, err := url.ParseQuery(rawQuery)
	if err != nil {
		return false
	}

	if _, ok := expected["*"]; ok {
		return len(vals) > 0
	}

	for key := range expected {
		if _, ok := vals[key]; ok {
			return true
		}
	}

	return false
}

func bodyFieldMatches(expected []string, actual any) bool {
	if len(expected) == 0 {
		return true
	}

	for _, fieldName := range expected {
		parts := strings.Split(fieldName, ".")

		if traverseBody(actual, parts) {
			return true
		}
	}

	return false
}

func traverseBody(val any, fields []string) bool {
	field := fields[0]

	switch v := val.(type) {
	case map[string]any:
		obj, ok := v[field]
		if !ok {
			return false
		}

		return traverseBody(obj, fields[1:])
	case []any:
		for _, item := range v {
			if traverseBody(item, fields) {
				return true
			}
		}

		return false
	}

	return false
}

func (r HostRule) CheckHostRule(job Job, jobID, ruleID string) ([]Finding, error) {
	var findings []Finding

	switch r.Type {
	case RuleTypePath:
		findings = append(findings, r.EvaluatePath(job.Metadata(), jobID, ruleID)...)
	case RuleTypeHeader:
	case RuleTypeQuery:
	case RuleTypeBodyField:
	default:
		return nil, nil
	}

	return findings, nil
}

func (r HostRule) EvaluatePath(meta Metadata, jobID, ruleID string) []Finding {
	if meta.Path == "" {
		return nil
	}

	for _, pattern := range r.Match.Patterns {
		re := pattern.Regex

		if pattern.Target != TargetTypePath {
			continue
		}

		if re.MatchString(meta.Path) {
			return []Finding{
				{
					ID:        uuid.NewString(),
					JobID:     jobID,
					RuleID:    ruleID,
					Title:     r.Finding.Title,
					Message:   r.Finding.Message,
					CreatedAt: time.Now().UTC(),
				},
			}
		}
	}

	return nil
}
