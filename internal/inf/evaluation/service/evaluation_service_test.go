package service

import (
	"awesome/internal/inf/evaluation/config"
	"awesome/internal/inf/evaluation/models"
	"context"
	"sync"
	"testing"
	"time"
)

func TestNewEvaluationService(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	service := NewEvaluationService(cfg)

	if service == nil {
		t.Fatal("Expected service to be created, but got nil")
	}

	if service.config != cfg {
		t.Error("Expected config to be set correctly")
	}

	if service.metricsCalc == nil {
		t.Error("Expected metricsCalc to be initialized")
	}

	if service.perfMonitor == nil {
		t.Error("Expected perfMonitor to be initialized")
	}

	if service.stabilityTester == nil {
		t.Error("Expected stabilityTester to be initialized")
	}

	if service.reportGenerator == nil {
		t.Error("Expected reportGenerator to be initialized")
	}

	if service.complianceChecker == nil {
		t.Error("Expected complianceChecker to be initialized")
	}

	if service.tasks == nil {
		t.Error("Expected tasks map to be initialized")
	}

	if service.results == nil {
		t.Error("Expected results map to be initialized")
	}
}

func TestEvaluationService_CreateTask(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	service := NewEvaluationService(cfg)

	ctx := context.Background()
	req := &CreateTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
		ModelID:     "model-123",
		ModelName:   "Test Model",
		Priority:    1,
		Config:      map[string]interface{}{"key": "value"},
	}

	task, err := service.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if task.ID == "" {
		t.Error("Expected task ID to be generated")
	}

	if task.Name != req.Name {
		t.Errorf("Expected task name %s, got %s", req.Name, task.Name)
	}

	if task.Description != req.Description {
		t.Errorf("Expected task description %s, got %s", req.Description, task.Description)
	}

	if task.ModelID != req.ModelID {
		t.Errorf("Expected task model ID %s, got %s", req.ModelID, task.ModelID)
	}

	if task.ModelName != req.ModelName {
		t.Errorf("Expected task model name %s, got %s", req.ModelName, task.ModelName)
	}

	if task.Priority != req.Priority {
		t.Errorf("Expected task priority %d, got %d", req.Priority, task.Priority)
	}

	if task.Status != models.TestStatusPending {
		t.Errorf("Expected task status %s, got %s", models.TestStatusPending, task.Status)
	}

	if task.Progress != 0 {
		t.Errorf("Expected task progress 0, got %f", task.Progress)
	}
}

func TestEvaluationService_StartTask(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	cfg.EnableMonitoring = false // Disable monitoring for this test
	service := NewEvaluationService(cfg)

	ctx := context.Background()
	req := &CreateTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
		ModelID:     "model-123",
		ModelName:   "Test Model",
		Priority:    1,
		Config:      map[string]interface{}{},
	}

	task, err := service.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error creating task: %v", err)
	}

	err = service.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Unexpected error starting task: %v", err)
	}

	// Wait a bit for the task to start
	time.Sleep(50 * time.Millisecond)

	// Check task status
	retrievedTask, err := service.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Unexpected error getting task: %v", err)
	}

	if retrievedTask.Status != models.TestStatusRunning && retrievedTask.Status != models.TestStatusCompleted {
		t.Errorf("Expected task status to be running or completed, got %s", retrievedTask.Status)
	}

	if retrievedTask.StartedAt == nil {
		t.Error("Expected StartedAt to be set")
	}
}

func TestEvaluationService_StartTask_NotFound(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	service := NewEvaluationService(cfg)

	ctx := context.Background()
	err := service.StartTask(ctx, "non-existent-task")
	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}

func TestEvaluationService_StartTask_WrongStatus(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	cfg.EnableMonitoring = false
	service := NewEvaluationService(cfg)

	ctx := context.Background()
	req := &CreateTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
		ModelID:     "model-123",
		ModelName:   "Test Model",
		Priority:    1,
		Config:      map[string]interface{}{},
	}

	task, err := service.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error creating task: %v", err)
	}

	// Manually set status to running
	task.Status = models.TestStatusRunning

	err = service.StartTask(ctx, task.ID)
	if err == nil {
		t.Error("Expected error when starting task that is not pending")
	}
}

func TestEvaluationService_GetTask(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	service := NewEvaluationService(cfg)

	ctx := context.Background()
	req := &CreateTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
		ModelID:     "model-123",
		ModelName:   "Test Model",
		Priority:    1,
		Config:      map[string]interface{}{},
	}

	task, err := service.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	retrievedTask, err := service.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if retrievedTask.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, retrievedTask.ID)
	}

	if retrievedTask.Name != task.Name {
		t.Errorf("Expected task name %s, got %s", task.Name, retrievedTask.Name)
	}
}

func TestEvaluationService_GetTask_NotFound(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	service := NewEvaluationService(cfg)

	_, err := service.GetTask("non-existent-task")
	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}

func TestEvaluationService_CancelTask(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	cfg.EnableMonitoring = false
	service := NewEvaluationService(cfg)

	ctx := context.Background()
	req := &CreateTaskRequest{
		Name:        "Test Task",
		Description: "Test Description",
		ModelID:     "model-123",
		ModelName:   "Test Model",
		Priority:    1,
		Config:      map[string]interface{}{},
	}

	task, err := service.CreateTask(ctx, req)
	if err != nil {
		t.Fatalf("Unexpected error creating task: %v", err)
	}

	// Start the task
	err = service.StartTask(ctx, task.ID)
	if err != nil {
		t.Fatalf("Unexpected error starting task: %v", err)
	}

	// Wait a bit for the task to start
	time.Sleep(50 * time.Millisecond)

	// Cancel the task
	err = service.CancelTask(task.ID)
	if err != nil {
		t.Fatalf("Unexpected error cancelling task: %v", err)
	}

	retrievedTask, err := service.GetTask(task.ID)
	if err != nil {
		t.Fatalf("Unexpected error getting task: %v", err)
	}

	// Task should be cancelled or completed (if it finished before cancellation)
	if retrievedTask.Status != models.TestStatusCancelled && retrievedTask.Status != models.TestStatusCompleted {
		t.Errorf("Expected task status to be cancelled or completed, got %s", retrievedTask.Status)
	}
}

func TestEvaluationService_CancelTask_NotFound(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	service := NewEvaluationService(cfg)

	err := service.CancelTask("non-existent-task")
	if err == nil {
		t.Error("Expected error for non-existent task")
	}
}

func TestEvaluationService_ListTasks(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	service := NewEvaluationService(cfg)

	ctx := context.Background()

	// Create multiple tasks
	for i := 0; i < 5; i++ {
		req := &CreateTaskRequest{
			Name:        "Test Task",
			Description: "Test Description",
			ModelID:     "model-123",
			ModelName:   "Test Model",
			Priority:    1,
			Config:      map[string]interface{}{},
		}
		_, err := service.CreateTask(ctx, req)
		if err != nil {
			t.Fatalf("Unexpected error creating task: %v", err)
		}
	}

	// List all tasks
	allTasks := service.ListTasks("", 0)
	if len(allTasks) != 5 {
		t.Errorf("Expected 5 tasks, got %d", len(allTasks))
	}

	// List tasks with limit
	limitedTasks := service.ListTasks("", 3)
	if len(limitedTasks) != 3 {
		t.Errorf("Expected 3 tasks with limit, got %d", len(limitedTasks))
	}

	// List tasks by status
	pendingTasks := service.ListTasks(models.TestStatusPending, 0)
	if len(pendingTasks) != 5 {
		t.Errorf("Expected 5 pending tasks, got %d", len(pendingTasks))
	}
}

func TestEvaluationService_ConcurrentAccess(t *testing.T) {
	cfg := config.DefaultEvaluationConfig()
	service := NewEvaluationService(cfg)

	ctx := context.Background()
	var wg sync.WaitGroup
	numGoroutines := 10

	// Test concurrent task creation
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := &CreateTaskRequest{
				Name:        "Test Task",
				Description: "Test Description",
				ModelID:     "model-123",
				ModelName:   "Test Model",
				Priority:    1,
				Config:      map[string]interface{}{},
			}
			_, err := service.CreateTask(ctx, req)
			if err != nil {
				t.Errorf("Unexpected error creating task: %v", err)
			}
		}(i)
	}

	wg.Wait()

	tasks := service.ListTasks("", 0)
	if len(tasks) != numGoroutines {
		t.Errorf("Expected %d tasks, got %d", numGoroutines, len(tasks))
	}
}

func TestGenerateTaskIDV1(t *testing.T) {
	id1 := generateTaskIDV1()
	id2 := generateTaskIDV1()

	if id1 == "" {
		t.Error("Expected non-empty task ID")
	}

	if id2 == "" {
		t.Error("Expected non-empty task ID")
	}

	// IDs should be different (based on timestamp)
	if id1 == id2 {
		t.Error("Expected different task IDs")
	}
}

func TestGenerateResultIDV1(t *testing.T) {
	id1 := generateResultIDV1()
	id2 := generateResultIDV1()

	if id1 == "" {
		t.Error("Expected non-empty result ID")
	}

	if id2 == "" {
		t.Error("Expected non-empty result ID")
	}

	// IDs should be different (based on timestamp)
	if id1 == id2 {
		t.Error("Expected different result IDs")
	}
}
