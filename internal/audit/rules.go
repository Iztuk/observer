package audit

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

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

func (r UpstreamFailureRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
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

func (r UpstreamTimeoutRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
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

type RequestPathDoesNotExistRule struct{}

func (r RequestPathDoesNotExistRule) ID() RuleID {
	return RuleRequestPathDoesNotExist
}

func (r RequestPathDoesNotExistRule) Title() string {
	return "Request path does not exist"
}

func (r RequestPathDoesNotExistRule) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestPathDoesNotExistRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	_, found := ctx.Contracts.FindPathItem(
		requestJob.Meta.Host,
		requestJob.Meta.Path,
	)

	if found {
		return nil, nil
	}

	return []Finding{
		{
			ID:        uuid.NewString(),
			JobID:     jobID,
			RuleID:    string(r.ID()),
			Title:     r.Title(),
			Message:   fmt.Sprintf("Request path %q is not defined in the API contract.", requestJob.Meta.Path),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type RequestMethodNotAllowedRule struct{}

func (r RequestMethodNotAllowedRule) ID() RuleID {
	return RuleRequestMethodNotAllowed
}

func (r RequestMethodNotAllowedRule) Title() string {
	return "Request method not allowed"
}

func (r RequestMethodNotAllowedRule) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestMethodNotAllowedRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	_, found := ctx.Contracts.FindMethod(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
	)

	if found {
		return nil, nil
	}

	return []Finding{
		{
			ID:        uuid.NewString(),
			JobID:     jobID,
			RuleID:    string(r.ID()),
			Title:     r.Title(),
			Message:   fmt.Sprintf("Request method %q is not defined in the API contract.", requestJob.Meta.Method),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type RequestContentTypeNotAllowed struct{}

func (r RequestContentTypeNotAllowed) ID() RuleID {
	return RuleRequestContentTypeNotAllowed
}

func (r RequestContentTypeNotAllowed) Title() string {
	return "Request content type not allowed"
}

func (r RequestContentTypeNotAllowed) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestContentTypeNotAllowed) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	contentType := requestJob.Headers.Get("Content-Type")

	_, applies, found := ctx.Contracts.FindContentType(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
		contentType,
	)

	if (applies && found) || !applies {
		return nil, nil
	}

	return []Finding{
		{
			ID:     uuid.NewString(),
			JobID:  jobID,
			RuleID: string(r.ID()),
			Title:  r.Title(),
			Message: fmt.Sprintf(
				"Request content type %q is not allowed for %s %s according to the API contract.",
				requestJob.Headers.Get("Content-Type"),
				requestJob.Meta.Method,
				requestJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type RequestBodyMissing struct{}

func (r RequestBodyMissing) ID() RuleID {
	return RuleRequestBodyMissing
}

func (r RequestBodyMissing) Title() string {
	return "Request body missing"
}

func (r RequestBodyMissing) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestBodyMissing) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	body, found := ctx.Contracts.FindBody(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
	)

	if !found {
		return nil, nil
	}

	if body != nil && body.Required && len(requestJob.Body) == 0 {
		return []Finding{
			{
				ID:     uuid.NewString(),
				JobID:  jobID,
				RuleID: string(r.ID()),
				Title:  r.Title(),
				Message: fmt.Sprintf(
					"Request body is required for %s %s according to the API contract, but the request body was empty.",
					requestJob.Meta.Method,
					requestJob.Meta.Path,
				),
				CreatedAt: time.Now().UTC(),
			},
		}, nil
	}

	return nil, nil
}

type RequestBodyNotAllowed struct{}

func (r RequestBodyNotAllowed) ID() RuleID {
	return RuleRequestBodyNotAllowed
}

func (r RequestBodyNotAllowed) Title() string {
	return "Request body not allowed"
}

func (r RequestBodyNotAllowed) AppliesTo() []JobType {
	return []JobType{RequestJobType}
}

func (r RequestBodyNotAllowed) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	requestJob, ok := job.(*RequestJob)
	if !ok {
		return nil, nil
	}

	body, found := ctx.Contracts.FindBody(
		requestJob.Meta.Host,
		requestJob.Meta.Method,
		requestJob.Meta.Path,
	)

	if !found {
		return nil, nil
	}

	if body != nil {
		return nil, nil
	}

	if len(requestJob.Body) == 0 {
		return nil, nil
	}

	return []Finding{
		{
			ID:     uuid.NewString(),
			JobID:  jobID,
			RuleID: string(r.ID()),
			Title:  r.Title(),
			Message: fmt.Sprintf(
				"Request body not allowed for %s %s according to the API contract, but the request body was not empty.",
				requestJob.Meta.Method,
				requestJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}
