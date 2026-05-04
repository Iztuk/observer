package rules

import (
	"cf-observer/internal/audit"
	"time"

	"github.com/google/uuid"
)

type RuleID string

const (
	RuleProxyUpstreamFailure RuleID = "proxy.upstream_failure"
	RuleProxyUpstreamTimeout RuleID = "proxy.upstream_host"
)

type Rule interface {
	ID() RuleID
	Title() string
	AppliesTo() []audit.JobType
	Check(job audit.Job, jobID string) ([]audit.Finding, error)
}

type RuleEngine struct {
	rules []Rule
}

func NewRuleEngine(rules []Rule) *RuleEngine {
	return &RuleEngine{
		rules: rules,
	}
}

func (e *RuleEngine) Evaluate(job audit.Job, jobID string) ([]audit.Finding, error) {
	var findings []audit.Finding

	for _, rule := range e.rules {
		if !ruleApplies(rule, job.JobType()) {
			continue
		}

		ruleFindings, err := rule.Check(job, jobID)
		if err != nil {
			return nil, err
		}

		findings = append(findings, ruleFindings...)
	}

	return findings, nil
}

func ruleApplies(rule Rule, jobType audit.JobType) bool {
	for _, supported := range rule.AppliesTo() {
		if supported == jobType {
			return true
		}
	}

	return false
}

type UpstreamFailureRule struct{}

func (r UpstreamFailureRule) ID() RuleID {
	return RuleProxyUpstreamFailure
}

func (r UpstreamFailureRule) Title() string {
	return "Upstream request failed"
}

func (r UpstreamFailureRule) AppliesTo() []audit.JobType {
	return []audit.JobType{audit.FailureJobType}
}

func (r UpstreamFailureRule) Check(job audit.Job, jobID string) ([]audit.Finding, error) {
	failureJob, ok := job.(*audit.FailureJob)
	if !ok {
		return nil, nil
	}

	if failureJob.Meta.Status == 504 {
		return nil, nil
	}

	return []audit.Finding{
		{
			ID:        uuid.NewString(),
			JobID:     jobID,
			RuleID:    string(r.ID()),
			Title:     r.Title(),
			Message:   failureJob.Error,
			CreatedAt: time.Now().UTC(),
		}}, nil
}

type UpstreamTimeoutRule struct{}

func (r UpstreamTimeoutRule) ID() RuleID {
	return RuleProxyUpstreamTimeout
}

func (r UpstreamTimeoutRule) Title() string {
	return "Upstream request timed out"
}

func (r UpstreamTimeoutRule) AppliesTo() []audit.JobType {
	return []audit.JobType{audit.FailureJobType}
}

func (r UpstreamTimeoutRule) Check(job audit.Job, jobID string) ([]audit.Finding, error) {
	failureJob, ok := job.(*audit.FailureJob)
	if !ok {
		return nil, nil
	}

	if failureJob.Meta.Status != 504 {
		return nil, nil
	}

	return []audit.Finding{
		{
			ID:        uuid.NewString(),
			JobID:     jobID,
			RuleID:    string(r.ID()),
			Title:     r.Title(),
			Message:   "The upstream service did not respond before the configured timeout.",
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}
