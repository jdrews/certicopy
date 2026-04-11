package core

import (
	"sync"

	"github.com/jdrews/certicopy/internal/models"
)

// TransferQueue manages a list of transfer jobs
type TransferQueue struct {
	jobs []*models.TransferJob
	mu   sync.RWMutex
}

// NewTransferQueue creates a new empty queue
func NewTransferQueue() *TransferQueue {
	return &TransferQueue{
		jobs: make([]*models.TransferJob, 0),
	}
}

// Add adds a job to the queue
func (q *TransferQueue) Add(job *models.TransferJob) {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = append(q.jobs, job)
}

// Peek returns the first pending/in-progress job in the queue, or nil if none
func (q *TransferQueue) Peek() *models.TransferJob {
	q.mu.RLock()
	defer q.mu.RUnlock()
	for _, job := range q.jobs {
		if job.Status == models.StatusPending || job.Status == models.StatusInProgress {
			return job
		}
	}
	return nil
}

// Pop removes and returns the first job in the queue, or nil if empty
func (q *TransferQueue) Pop() *models.TransferJob {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.jobs) == 0 {
		return nil
	}
	job := q.jobs[0]
	q.jobs = q.jobs[1:]
	return job
}

// GetAll returns all jobs in the queue
func (q *TransferQueue) GetAll() []*models.TransferJob {
	q.mu.RLock()
	defer q.mu.RUnlock()
	// Return a copy to avoid race conditions
	jobs := make([]*models.TransferJob, len(q.jobs))
	copy(jobs, q.jobs)
	return jobs
}

// Remove removes a job by ID
func (q *TransferQueue) Remove(id string) bool {
	q.mu.Lock()
	defer q.mu.Unlock()
	for i, job := range q.jobs {
		if job.ID == id {
			q.jobs = append(q.jobs[:i], q.jobs[i+1:]...)
			return true
		}
	}
	return false
}

// Clear removes all jobs
func (q *TransferQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.jobs = make([]*models.TransferJob, 0)
}
