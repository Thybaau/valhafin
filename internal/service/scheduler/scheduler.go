package scheduler

import (
	"context"
	"log"
	"sync"
	"time"
	"valhafin/internal/service/price"
	"valhafin/internal/service/scraper/types"
)

// SyncService defines the interface for synchronization operations
type SyncService interface {
	SyncAccount(accountID string) (*types.SyncResult, error)
	SyncAllAccounts() ([]types.SyncResult, error)
}

// Task represents a scheduled task
type Task struct {
	Name     string
	Interval time.Duration
	Fn       func() error
}

// Scheduler manages periodic tasks
type Scheduler struct {
	tasks        []Task
	ctx          context.Context
	cancel       context.CancelFunc
	wg           sync.WaitGroup
	priceService price.Service
	syncService  SyncService
}

// NewScheduler creates a new scheduler instance
func NewScheduler(priceService price.Service, syncService SyncService) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		tasks:        make([]Task, 0),
		ctx:          ctx,
		cancel:       cancel,
		priceService: priceService,
		syncService:  syncService,
	}
}

// AddTask adds a new task to the scheduler
func (s *Scheduler) AddTask(name string, interval time.Duration, fn func() error) {
	s.tasks = append(s.tasks, Task{
		Name:     name,
		Interval: interval,
		Fn:       fn,
	})
}

// Start begins executing all scheduled tasks
func (s *Scheduler) Start() {
	log.Println("📅 Scheduler starting...")

	// Add default tasks
	s.addDefaultTasks()

	// Start each task in its own goroutine
	for _, task := range s.tasks {
		s.wg.Add(1)
		go s.runTask(task)
	}

	log.Printf("📅 Scheduler started with %d tasks", len(s.tasks))
}

// Stop gracefully stops all scheduled tasks
func (s *Scheduler) Stop() {
	log.Println("📅 Scheduler stopping...")
	s.cancel()
	s.wg.Wait()
	log.Println("📅 Scheduler stopped")
}

// runTask executes a task at the specified interval
func (s *Scheduler) runTask(task Task) {
	defer s.wg.Done()

	ticker := time.NewTicker(task.Interval)
	defer ticker.Stop()

	log.Printf("📅 Task '%s' scheduled to run every %s", task.Name, task.Interval)

	// Run immediately on start
	if err := task.Fn(); err != nil {
		log.Printf("❌ Task '%s' failed: %v", task.Name, err)
	} else {
		log.Printf("✅ Task '%s' completed successfully", task.Name)
	}

	for {
		select {
		case <-s.ctx.Done():
			log.Printf("📅 Task '%s' stopped", task.Name)
			return
		case <-ticker.C:
			log.Printf("📅 Running task '%s'", task.Name)
			if err := task.Fn(); err != nil {
				log.Printf("❌ Task '%s' failed: %v", task.Name, err)
			} else {
				log.Printf("✅ Task '%s' completed successfully", task.Name)
			}
		}
	}
}

// addDefaultTasks adds the default scheduled tasks
func (s *Scheduler) addDefaultTasks() {
	// Task 1: Update asset prices every hour
	s.AddTask("update_prices", 1*time.Hour, func() error {
		log.Println("💰 Updating asset prices...")
		if err := s.priceService.UpdateAllPrices(); err != nil {
			log.Printf("❌ Failed to update prices: %v", err)
			return err
		}
		log.Println("💰 Asset prices updated successfully")
		return nil
	})

	// Task 2: Sync all accounts daily
	s.AddTask("sync_accounts", 24*time.Hour, func() error {
		log.Println("🔄 Syncing all accounts...")
		results, err := s.syncService.SyncAllAccounts()
		if err != nil {
			log.Printf("❌ Failed to sync accounts: %v", err)
			return err
		}

		// Log summary
		successCount := 0
		failCount := 0
		for _, result := range results {
			if result.Error == "" {
				successCount++
			} else {
				failCount++
			}
		}

		log.Printf("🔄 Account sync completed: %d successful, %d failed", successCount, failCount)
		return nil
	})
}
