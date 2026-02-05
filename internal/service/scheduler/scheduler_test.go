package scheduler

import (
	"errors"
	"sync"
	"testing"
	"time"
	"valhafin/internal/domain/models"
	"valhafin/internal/service/scraper/types"
)

// mockPriceService is a mock implementation of the price.Service interface
type mockPriceService struct {
	updateAllPricesCalled int
	updateAllPricesError  error
	mu                    sync.Mutex
}

func (m *mockPriceService) GetCurrentPrice(isin string) (*models.AssetPrice, error) {
	return nil, nil
}

func (m *mockPriceService) GetPriceHistory(isin string, startDate, endDate time.Time) ([]models.AssetPrice, error) {
	return nil, nil
}

func (m *mockPriceService) UpdateAllPrices() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.updateAllPricesCalled++
	return m.updateAllPricesError
}

func (m *mockPriceService) UpdateAssetPrice(isin string) error {
	return nil
}

func (m *mockPriceService) getUpdateAllPricesCalled() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updateAllPricesCalled
}

// mockSyncService is a mock implementation of the sync.Service
type mockSyncService struct {
	syncAllAccountsCalled int
	syncAllAccountsError  error
	syncResults           []types.SyncResult
	mu                    sync.Mutex
}

func (m *mockSyncService) SyncAccount(accountID string) (*types.SyncResult, error) {
	return &types.SyncResult{AccountID: accountID}, nil
}

func (m *mockSyncService) SyncAllAccounts() ([]types.SyncResult, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.syncAllAccountsCalled++
	return m.syncResults, m.syncAllAccountsError
}

func (m *mockSyncService) getSyncAllAccountsCalled() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.syncAllAccountsCalled
}

// TestSchedulerTaskExecution tests that tasks are executed at the correct intervals
func TestSchedulerTaskExecution(t *testing.T) {
	// Create mock services
	mockPrice := &mockPriceService{}
	mockSync := &mockSyncService{
		syncResults: []types.SyncResult{
			{AccountID: "test-1", Error: ""},
			{AccountID: "test-2", Error: ""},
		},
	}

	// Create scheduler with short intervals for testing
	scheduler := NewScheduler(mockPrice, mockSync)

	// Override default tasks with test tasks
	scheduler.tasks = []Task{}

	// Add a task that runs every 100ms
	taskExecutionCount := 0
	var taskMu sync.Mutex
	scheduler.AddTask("test_task", 100*time.Millisecond, func() error {
		taskMu.Lock()
		taskExecutionCount++
		taskMu.Unlock()
		return nil
	})

	// Start scheduler
	scheduler.Start()

	// Wait for task to execute multiple times
	time.Sleep(350 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify task was executed multiple times
	taskMu.Lock()
	count := taskExecutionCount
	taskMu.Unlock()

	// Task should run immediately on start, then at 100ms, 200ms, 300ms
	// So we expect at least 3-4 executions
	if count < 3 {
		t.Errorf("Expected task to execute at least 3 times, got %d", count)
	}

	if count > 5 {
		t.Errorf("Expected task to execute at most 5 times, got %d (timing issue?)", count)
	}
}

// TestSchedulerDefaultTasks tests that default tasks are added and executed
func TestSchedulerDefaultTasks(t *testing.T) {
	// Create mock services
	mockPrice := &mockPriceService{}
	mockSync := &mockSyncService{
		syncResults: []types.SyncResult{
			{AccountID: "test-1", Error: ""},
		},
	}

	// Create scheduler
	scheduler := NewScheduler(mockPrice, mockSync)

	// Start scheduler (this will add default tasks and run them immediately)
	scheduler.Start()

	// Give tasks time to execute once (they run immediately on start)
	time.Sleep(100 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify that both default tasks were executed at least once
	priceCallCount := mockPrice.getUpdateAllPricesCalled()
	syncCallCount := mockSync.getSyncAllAccountsCalled()

	if priceCallCount < 1 {
		t.Errorf("Expected UpdateAllPrices to be called at least once, got %d", priceCallCount)
	}

	if syncCallCount < 1 {
		t.Errorf("Expected SyncAllAccounts to be called at least once, got %d", syncCallCount)
	}
}

// TestSchedulerTaskInterval tests that tasks respect their configured intervals
func TestSchedulerTaskInterval(t *testing.T) {
	// Create mock services
	mockPrice := &mockPriceService{}
	mockSync := &mockSyncService{}

	// Create scheduler
	scheduler := NewScheduler(mockPrice, mockSync)
	scheduler.tasks = []Task{}

	// Track execution times
	var executionTimes []time.Time
	var timeMu sync.Mutex

	interval := 150 * time.Millisecond
	scheduler.AddTask("interval_test", interval, func() error {
		timeMu.Lock()
		executionTimes = append(executionTimes, time.Now())
		timeMu.Unlock()
		return nil
	})

	// Start scheduler
	startTime := time.Now()
	scheduler.Start()

	// Wait for multiple executions
	time.Sleep(500 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify intervals between executions
	timeMu.Lock()
	times := make([]time.Time, len(executionTimes))
	copy(times, executionTimes)
	timeMu.Unlock()

	if len(times) < 3 {
		t.Fatalf("Expected at least 3 executions, got %d", len(times))
	}

	// Check that first execution happened immediately (within 50ms of start)
	firstDelay := times[0].Sub(startTime)
	if firstDelay > 50*time.Millisecond {
		t.Errorf("Expected first execution to be immediate, but it took %v", firstDelay)
	}

	// Check intervals between subsequent executions
	for i := 1; i < len(times); i++ {
		actualInterval := times[i].Sub(times[i-1])
		// Allow 50ms tolerance for timing variations
		if actualInterval < interval-50*time.Millisecond || actualInterval > interval+50*time.Millisecond {
			t.Errorf("Expected interval ~%v between executions %d and %d, got %v",
				interval, i-1, i, actualInterval)
		}
	}
}

// TestSchedulerErrorHandling tests that scheduler continues running even if tasks fail
func TestSchedulerErrorHandling(t *testing.T) {
	// Create mock services that return errors
	mockPrice := &mockPriceService{
		updateAllPricesError: errors.New("mock price error"),
	}
	mockSync := &mockSyncService{
		syncAllAccountsError: errors.New("mock sync error"),
	}

	// Create scheduler
	scheduler := NewScheduler(mockPrice, mockSync)

	// Start scheduler
	scheduler.Start()

	// Wait for tasks to execute
	time.Sleep(100 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify that tasks were called despite errors
	priceCallCount := mockPrice.getUpdateAllPricesCalled()
	syncCallCount := mockSync.getSyncAllAccountsCalled()

	if priceCallCount < 1 {
		t.Errorf("Expected UpdateAllPrices to be called despite error, got %d calls", priceCallCount)
	}

	if syncCallCount < 1 {
		t.Errorf("Expected SyncAllAccounts to be called despite error, got %d calls", syncCallCount)
	}
}

// TestSchedulerGracefulShutdown tests that scheduler stops gracefully
func TestSchedulerGracefulShutdown(t *testing.T) {
	// Create mock services
	mockPrice := &mockPriceService{}
	mockSync := &mockSyncService{}

	// Create scheduler
	scheduler := NewScheduler(mockPrice, mockSync)
	scheduler.tasks = []Task{}

	// Add a long-running task
	taskStarted := make(chan bool, 1)
	taskCompleted := make(chan bool, 1)

	scheduler.AddTask("long_task", 1*time.Hour, func() error {
		taskStarted <- true
		time.Sleep(200 * time.Millisecond)
		taskCompleted <- true
		return nil
	})

	// Start scheduler
	scheduler.Start()

	// Wait for task to start
	<-taskStarted

	// Stop scheduler (should wait for task to complete)
	stopStart := time.Now()
	scheduler.Stop()
	stopDuration := time.Since(stopStart)

	// Verify task completed
	select {
	case <-taskCompleted:
		// Task completed successfully
	case <-time.After(1 * time.Second):
		t.Error("Task did not complete after scheduler stop")
	}

	// Verify stop waited for task (should take ~200ms)
	if stopDuration < 150*time.Millisecond {
		t.Errorf("Stop returned too quickly (%v), may not have waited for task", stopDuration)
	}
}

// TestSchedulerMultipleTasks tests that multiple tasks can run concurrently
func TestSchedulerMultipleTasks(t *testing.T) {
	// Create mock services
	mockPrice := &mockPriceService{}
	mockSync := &mockSyncService{}

	// Create scheduler
	scheduler := NewScheduler(mockPrice, mockSync)
	scheduler.tasks = []Task{}

	// Track execution counts for multiple tasks
	task1Count := 0
	task2Count := 0
	task3Count := 0
	var mu sync.Mutex

	scheduler.AddTask("task1", 50*time.Millisecond, func() error {
		mu.Lock()
		task1Count++
		mu.Unlock()
		return nil
	})

	scheduler.AddTask("task2", 75*time.Millisecond, func() error {
		mu.Lock()
		task2Count++
		mu.Unlock()
		return nil
	})

	scheduler.AddTask("task3", 100*time.Millisecond, func() error {
		mu.Lock()
		task3Count++
		mu.Unlock()
		return nil
	})

	// Start scheduler
	scheduler.Start()

	// Wait for tasks to execute
	time.Sleep(250 * time.Millisecond)

	// Stop scheduler
	scheduler.Stop()

	// Verify all tasks executed
	mu.Lock()
	t1 := task1Count
	t2 := task2Count
	t3 := task3Count
	mu.Unlock()

	if t1 < 3 {
		t.Errorf("Expected task1 to execute at least 3 times, got %d", t1)
	}

	if t2 < 2 {
		t.Errorf("Expected task2 to execute at least 2 times, got %d", t2)
	}

	if t3 < 2 {
		t.Errorf("Expected task3 to execute at least 2 times, got %d", t3)
	}

	// Verify tasks ran at different rates (task1 should run more than task3)
	if t1 <= t3 {
		t.Errorf("Expected task1 (50ms) to run more than task3 (100ms), got task1=%d, task3=%d", t1, t3)
	}
}
