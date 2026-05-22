package audit

import (
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

	if !methodMatches(r.Match.Methods, meta.Method) {
		return false
	}

	if !pathMatchesAny(r.Match.Paths, meta.Path) {
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

func (r HostRule) CheckHostRule(job Job, jobID, ruleID string) ([]Finding, error) {
	var findings []Finding

	switch r.Type {
	case RuleTypePath:
		findings = append(findings, r.EvaluatePath(job.Metadata(), jobID, ruleID)...)
	case RuleTypeQuery:
	case RuleTypeHeader:
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
