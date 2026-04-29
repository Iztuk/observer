package audit

import (
	"fmt"
	"log"
	"runtime/debug"
	"sync"
)

type Job interface {
	JobType() JobType
	Process()
}

type Queue struct {
	jobs chan Job
	mu   sync.RWMutex
	done bool
	once sync.Once
}

func (r RequestJob) JobType() JobType {
	return RequestJobType
}

func (r RequestJob) Process() {
	fmt.Printf("\nProcessing Request Job: %v", r)
}

func (r ResponseJob) JobType() JobType {
	return ResponseJobType
}

func (r ResponseJob) Process() {
	fmt.Printf("\nProcessing Response Job: %v", r)
}

func (r FailureJob) JobType() JobType {
	return FailureJobType
}

func (r FailureJob) Process() {
	fmt.Printf("\nProcessing Failure Job: %v", r)
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

func (q *Queue) StartWorkers(count int, logger *log.Logger) *sync.WaitGroup {
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

					if err := ProcessJob(job); err != nil {
						logger.Printf("audit worker %d failed to process job: %v", workerID, err)
					}
				}()
			}

			logger.Printf("audit worker %d queue closed", workerID)
		}(i)
	}

	return &wg
}

func ProcessJob(job Job) error {
	if job == nil {
		return fmt.Errorf("nil audit job")
	}

	job.Process()
	return nil
}

func (q *Queue) Close() {
	q.once.Do(func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		q.done = true
		close(q.jobs)
	})
}
