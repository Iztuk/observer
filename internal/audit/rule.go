package audit

import (
	"time"

	"github.com/google/uuid"
)

type RuleID string

const (
	RuleProxyUpstreamFailure RuleID = "proxy.upstream_failure"
	RuleProxyUpstreamTimeout RuleID = "proxy.upstream_timeout"
)

type Rule interface {
	ID() RuleID
	Title() string
	AppliesTo() []JobType
	Check(job Job, jobID string) ([]Finding, error)
}

type RuleEngine struct {
	rules []Rule
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: getRules(),
	}
}

func getRules() []Rule {
	return []Rule{
		UpstreamFailureRule{},
		UpstreamTimeoutRule{},
	}
}

func (e *RuleEngine) Evaluate(job Job, jobID string) ([]Finding, error) {
	var findings []Finding

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

func ruleApplies(rule Rule, jobType JobType) bool {
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

func (r UpstreamFailureRule) AppliesTo() []JobType {
	return []JobType{FailureJobType}
}

func (r UpstreamFailureRule) Check(job Job, jobID string) ([]Finding, error) {
	failureJob, ok := job.(*FailureJob)
	if !ok {
		return nil, nil
	}

	if failureJob.Meta.Status == 504 {
		return nil, nil
	}

	return []Finding{
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

func (r UpstreamTimeoutRule) AppliesTo() []JobType {
	return []JobType{FailureJobType}
}

func (r UpstreamTimeoutRule) Check(job Job, jobID string) ([]Finding, error) {
	failureJob, ok := job.(*FailureJob)
	if !ok {
		return nil, nil
	}

	if failureJob.Meta.Status != 504 {
		return nil, nil
	}

	return []Finding{
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
