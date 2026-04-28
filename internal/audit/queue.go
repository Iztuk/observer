package audit

import (
	"fmt"
	"log"
	"sync"
)

type Job interface {
	JobType() JobType
	Process()
}

type Queue struct {
	jobs chan Job
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

func worker(id int, queue <-chan Job, wg *sync.WaitGroup) {
	defer wg.Done()

	for job := range queue {
		fmt.Printf("[Worker %d] Handling %s", id, job.JobType())
		job.Process()
	}
}

func NewQueue(size int) *Queue {
	return &Queue{
		jobs: make(chan Job, size),
	}
}

func (q *Queue) TryEnqueue(job Job) bool {
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
				logger.Printf("audit worker %d handling %s", workerID, job.JobType())

				if err := ProcessJob(job); err != nil {
					logger.Printf("audit worker %d failed to process job: %v", workerID, err)
				}
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
	close(q.jobs)
}
