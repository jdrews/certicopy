package core

import (
	"sync"
	"testing"

	"github.com/jdrews/certicopy/internal/models"
)

func TestTransferQueue_Basic(t *testing.T) {
	q := NewTransferQueue()

	// Initial state
	if q.Peek() != nil {
		t.Error("Expected empty queue to return nil for Peek")
	}
	if q.Pop() != nil {
		t.Error("Expected empty queue to return nil for Pop")
	}

	// Add jobs
	job1 := &models.TransferJob{ID: "1", Status: models.StatusPending}
	job2 := &models.TransferJob{ID: "2", Status: models.StatusPending}
	q.Add(job1)
	q.Add(job2)

	// GetAll
	jobs := q.GetAll()
	if len(jobs) != 2 {
		t.Errorf("Expected 2 jobs in queue, got %d", len(jobs))
	}

	// Peek
	peeked := q.Peek()
	if peeked == nil || peeked.ID != "1" {
		t.Errorf("Expected peek to return job 1, got %v", peeked)
	}

	// Pop
	popped := q.Pop()
	if popped == nil || popped.ID != "1" {
		t.Errorf("Expected pop to return job 1, got %v", popped)
	}

	if len(q.GetAll()) != 1 {
		t.Errorf("Expected 1 job remaining after pop, got %d", len(q.GetAll()))
	}
}

func TestTransferQueue_PeekFilters(t *testing.T) {
	q := NewTransferQueue()

	job1 := &models.TransferJob{ID: "1", Status: models.StatusSuccess}
	job2 := &models.TransferJob{ID: "2", Status: models.StatusInProgress}
	job3 := &models.TransferJob{ID: "3", Status: models.StatusPending}

	q.Add(job1)
	q.Add(job2)
	q.Add(job3)

	// Peek should skip job1 because it's Completed
	peeked := q.Peek()
	if peeked == nil || peeked.ID != "2" {
		t.Errorf("Expected peek to return job 2 (InProgress), got %v", peeked)
	}
}

func TestTransferQueue_RemoveAndClear(t *testing.T) {
	q := NewTransferQueue()

	job1 := &models.TransferJob{ID: "1"}
	job2 := &models.TransferJob{ID: "2"}
	q.Add(job1)
	q.Add(job2)

	// Remove non-existent
	if q.Remove("3") {
		t.Error("Expected Remove for non-existent ID to return false")
	}

	// Remove existent
	if !q.Remove("1") {
		t.Error("Expected Remove for existent ID to return true")
	}

	if len(q.GetAll()) != 1 {
		t.Errorf("Expected 1 job remaining after removal, got %d", len(q.GetAll()))
	}

	// Clear
	q.Clear()
	if len(q.GetAll()) != 0 {
		t.Errorf("Expected 0 jobs after clear, got %d", len(q.GetAll()))
	}
}

func TestTransferQueue_Concurrency(t *testing.T) {
	q := NewTransferQueue()
	var wg sync.WaitGroup
	const workers = 10
	const itemsPerWorker = 100

	// Concurrent Add
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(w int) {
			defer wg.Done()
			for j := 0; j < itemsPerWorker; j++ {
				q.Add(&models.TransferJob{ID: "task"})
			}
		}(i)
	}
	wg.Wait()

	all := q.GetAll()
	if len(all) != workers*itemsPerWorker {
		t.Errorf("Expected %d items, got %d", workers*itemsPerWorker, len(all))
	}

	// Concurrent Pop and Remove
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < itemsPerWorker/2; j++ {
				q.Pop()
			}
		}()
	}
	wg.Wait()

	remaining := q.GetAll()
	expectedRemaining := (workers * itemsPerWorker) - (workers * itemsPerWorker / 2)
	if len(remaining) != expectedRemaining {
		t.Errorf("Expected %d items remaining, got %d", expectedRemaining, len(remaining))
	}
}
