package audit

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"

	"github.com/google/uuid"
)

type Job interface {
	JobType() JobType
	Process(e *RuleEngine) error
}

type Queue struct {
	jobs chan Job
	mu   sync.RWMutex
	done bool
	once sync.Once
}

func (r *RequestJob) JobType() JobType {
	return RequestJobType
}

func (r *RequestJob) Process(engine *RuleEngine) error {
	jobID := uuid.NewString()

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		return nil
	}

	return DatabaseStore.SaveAuditResult(r, jobID, findings)
}

func (r *ResponseJob) JobType() JobType {
	return ResponseJobType
}

func (r *ResponseJob) Process(engine *RuleEngine) error {
	jobID := uuid.NewString()

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		return nil
	}

	return DatabaseStore.SaveAuditResult(r, jobID, findings)
}

func (r *FailureJob) JobType() JobType {
	return FailureJobType
}

func (r *FailureJob) Process(engine *RuleEngine) error {
	jobID := uuid.NewString()

	findings, err := engine.Evaluate(r, jobID)
	if err != nil {
		return err
	}

	if len(findings) == 0 {
		return nil
	}

	return DatabaseStore.SaveAuditResult(r, jobID, findings)
}

func NewQueue(size int) *Queue {
	return &Queue{
		jobs: make(chan Job, size),
	}
}

func (q *Queue) TryEnqueue(job Job) bool {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.done {
		return false
	}

	if job == nil {
		return false
	}

	select {
	case q.jobs <- job:
		return true
	default:
		return false
	}
}

func (q *Queue) StartWorkers(count int, logger *log.Logger, engine *RuleEngine) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := 0; i < count; i++ {
		wg.Add(1)

		go func(workerID int) {
			defer wg.Done()

			for job := range q.jobs {
				func() {
					defer func() {
						if r := recover(); r != nil {
							logger.Printf(
								"audit worker %d panic: %v\n%s",
								workerID,
								r,
								debug.Stack(),
							)
						}
					}()

					if err := ProcessJob(job, engine); err != nil {
						logger.Printf("audit worker %d failed to process job: %v", workerID, err)
					}
				}()
			}

			logger.Printf("audit worker %d queue closed", workerID)
		}(i)
	}

	return &wg
}

func ProcessJob(job Job, engine *RuleEngine) error {
	if job == nil {
		return fmt.Errorf("nil audit job")
	}

	return job.Process(engine)
}

func (q *Queue) Close() {
	q.once.Do(func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.done = true
		close(q.jobs)
	})
}
