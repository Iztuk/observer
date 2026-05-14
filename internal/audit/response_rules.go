package audit

import (
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
)

type ResponseStatusCodeRule struct{}

func (r ResponseStatusCodeRule) ID() RuleID {
	return RuleResponseStatusCodeNotDefined
}

func (r ResponseStatusCodeRule) Title() string {
	return "Response status code not defined"
}

func (r ResponseStatusCodeRule) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseStatusCodeRule) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	responseJob, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	op, found := ctx.Contracts.FindMethod(
		responseJob.Meta.Host,
		responseJob.Meta.Method,
		responseJob.Meta.Path,
	)
	if !found {
		return nil, nil
	}

	status := strconv.Itoa(responseJob.Meta.Status)
	_, found = op.Responses[status]
	if found {
		return nil, nil
	}

	return []Finding{
		{
			ID:     uuid.NewString(),
			JobID:  jobID,
			RuleID: string(r.ID()),
			Title:  r.Title(),
			Message: fmt.Sprintf(
				"Response status code %d for %s %s is not defined in the API contract.",
				responseJob.Meta.Status,
				responseJob.Meta.Method,
				responseJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type ResponseContentTypeNotAllowed struct{}

func (r ResponseContentTypeNotAllowed) ID() RuleID {
	return RuleResponseContentTypeNotAllowed
}

func (r ResponseContentTypeNotAllowed) Title() string {
	return "Response content type not allowed"
}

func (r ResponseContentTypeNotAllowed) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseContentTypeNotAllowed) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	responseJob, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	ct := responseJob.Headers.Get("Content-Type")
	status := strconv.Itoa(responseJob.Meta.Status)

	_, applies, found := ctx.Contracts.FindResponseContentType(
		responseJob.Meta.Host,
		responseJob.Meta.Method,
		responseJob.Meta.Path,
		status,
		ct,
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
				"Response content type %q is not allowed for %s %s according to the API contract.",
				responseJob.Headers.Get("Content-Type"),
				responseJob.Meta.Method,
				responseJob.Meta.Path,
			),
			CreatedAt: time.Now().UTC(),
		},
	}, nil
}

type ResponseBodyMissing struct{}

func (r ResponseBodyMissing) ID() RuleID {
	return RuleResponseBodyMissing
}

func (r ResponseBodyMissing) Title() string {
	return "Response body missing"
}

func (r ResponseBodyMissing) AppliesTo() []JobType {
	return []JobType{ResponseJobType}
}

func (r ResponseBodyMissing) Check(ctx RuleContext, job Job, jobID string) ([]Finding, error) {
	responseJob, ok := job.(*ResponseJob)
	if !ok {
		return nil, nil
	}

	status := strconv.Itoa(responseJob.Meta.Status)
	body, found := ctx.Contracts.FindResponseBody(
		responseJob.Meta.Host,
		responseJob.Meta.Method,
		responseJob.Meta.Path,
		status,
	)

	if !found {
		return nil, nil
	}

	if len(body.Content) == 0 {
		return nil, nil
	}

	if len(responseJob.Body) == 0 {
		return []Finding{
			{
				ID:     uuid.NewString(),
				JobID:  jobID,
				RuleID: string(r.ID()),
				Title:  r.Title(),
				Message: fmt.Sprintf(
					"Response body is required for %s %s according to the API contract, but the request body was empty.",
					responseJob.Meta.Method,
					responseJob.Meta.Path,
				),
				CreatedAt: time.Now().UTC(),
			},
		}, nil
	}

	return nil, nil
}
