package audit

import (
	"fmt"
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
	fmt.Println("Processing Request Job: %v", r)
}

func (r ResponseJob) JobType() JobType {
	return ResponseJobType
}

func (r ResponseJob) Process() {
	fmt.Println("Processing Response Job: %v", r)
}

func (r FailureJob) JobType() JobType {
	return FailureJobType
}

func (r FailureJob) Process() {
	fmt.Println("Processing Failure Job: %v", r)
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

func (q *Queue) Enqueue(job Job) {
	q.jobs <- job
}

func (q *Queue) StartWorkers(n int) *sync.WaitGroup {
	var wg sync.WaitGroup

	for i := 0; i < n; i++ {
		wg.Add(1)
		go worker(i, q.jobs, &wg)
	}

	return &wg
}

func (q *Queue) Close() {
	close(q.jobs)
}
