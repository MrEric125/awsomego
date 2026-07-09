package pipeline

import (
	"awesome/internal/inf/evaluation/config"
	"awesome/internal/inf/evaluation/models"
	"context"
	"fmt"
	"sync"
	"time"
)

// Scheduler 调度器
type Scheduler struct {
	config     *config.EvaluationConfig
	pipelines  map[string]*Pipeline
	runs       map[string]*models.PipelineRun
	eventQueue chan Event
	notifyChan chan Notification
	mu         sync.RWMutex
	ctx        context.Context
	cancel     context.CancelFunc
	running    bool
}

// Event 触发事件
type Event struct {
	Type      string
	Source    string
	Timestamp time.Time
	Payload   map[string]interface{}
}

// Notification 通知
type Notification struct {
	Type      string // progress, complete, error
	TaskID    string
	Message   string
	Progress  float64
	Timestamp time.Time
	Details   map[string]interface{}
}

// Pipeline 流水线
type Pipeline struct {
	ID          string
	Config      *config.PipelineConfig
	Tasks       []TaskExecutor
	Triggers    []Trigger
	Status      models.TestStatus
	LastRunTime *time.Time
	NextRunTime *time.Time
}

// TaskExecutor 任务执行器接口
type TaskExecutor interface {
	Execute(ctx context.Context) error
	GetName() string
	GetProgress() float64
}

// Trigger 触发器接口
type Trigger interface {
	ShouldTrigger(event Event) bool
	GetType() string
}

// NewScheduler 创建调度器
func NewScheduler(cfg *config.EvaluationConfig) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		config:     cfg,
		pipelines:  make(map[string]*Pipeline),
		runs:       make(map[string]*models.PipelineRun),
		eventQueue: make(chan Event, 10000),
		notifyChan: make(chan Notification, 1000),
		ctx:        ctx,
		cancel:     cancel,
		running:    false,
	}
}

// Start 启动调度器
func (s *Scheduler) Start() error {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return fmt.Errorf("scheduler already running")
	}
	s.running = true
	s.mu.Unlock()

	// 启动事件处理协程
	go s.eventLoop()

	// 启动定时任务调度
	go s.scheduleLoop()

	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.cancel()
		s.running = false
	}
}

// eventLoop 事件处理循环
func (s *Scheduler) eventLoop() {
	for {
		select {
		case <-s.ctx.Done():
			return
		case event := <-s.eventQueue:
			s.handleEvent(event)
		}
	}
}

// scheduleLoop 定时调度循环
func (s *Scheduler) scheduleLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.checkScheduledPipelines()
		}
	}
}

// handleEvent 处理事件
func (s *Scheduler) handleEvent(event Event) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, pipeline := range s.pipelines {
		for _, trigger := range pipeline.Triggers {
			if trigger.ShouldTrigger(event) {
				go s.executePipeline(pipeline, "event", event.Source)
			}
		}
	}
}

// checkScheduledPipelines 检查定时任务
func (s *Scheduler) checkScheduledPipelines() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, pipeline := range s.pipelines {
		if pipeline.NextRunTime != nil && now.After(*pipeline.NextRunTime) {
			go s.executePipeline(pipeline, "scheduled", "scheduler")

			// 计算下次运行时间
			if pipeline.Config.Schedule != "" {
				next := s.calculateNextRun(pipeline.Config.Schedule)
				pipeline.NextRunTime = &next
			}
		}
	}
}

// executePipeline 执行流水线
func (s *Scheduler) executePipeline(pipeline *Pipeline, triggerType, triggeredBy string) {
	run := &models.PipelineRun{
		ID:          generateRunID(),
		PipelineID:  pipeline.ID,
		Name:        pipeline.Config.Name,
		TriggerType: triggerType,
		TriggeredBy: triggeredBy,
		Status:      models.TestStatusRunning,
		StartTime:   time.Now(),
		TaskIDs:     make([]string, 0),
		Progress:    0,
		Artifacts:   make(map[string]string),
		Metadata:    make(map[string]interface{}),
	}

	s.mu.Lock()
	s.runs[run.ID] = run
	s.mu.Unlock()

	// 发送开始通知
	s.sendNotification(Notification{
		Type:      "progress",
		TaskID:    run.ID,
		Message:   fmt.Sprintf("Pipeline %s started", pipeline.Config.Name),
		Progress:  0,
		Timestamp: time.Now(),
	})

	// 执行任务
	var lastErr error
	for i, task := range pipeline.Tasks {
		if err := task.Execute(s.ctx); err != nil {
			lastErr = err
			run.Status = models.TestStatusFailed
			run.Error = err.Error()

			if !pipeline.Config.RetryOnFailure {
				break
			}
		}

		run.Progress = float64(i+1) / float64(len(pipeline.Tasks)) * 100

		// 发送进度通知
		s.sendNotification(Notification{
			Type:      "progress",
			TaskID:    run.ID,
			Message:   fmt.Sprintf("Task %s completed", task.GetName()),
			Progress:  run.Progress,
			Timestamp: time.Now(),
		})
	}

	// 完成
	now := time.Now()
	run.EndTime = &now
	run.Duration = now.Sub(run.StartTime)

	if lastErr == nil {
		run.Status = models.TestStatusCompleted

		if pipeline.Config.NotifyOnComplete {
			s.sendNotification(Notification{
				Type:      "complete",
				TaskID:    run.ID,
				Message:   fmt.Sprintf("Pipeline %s completed successfully", pipeline.Config.Name),
				Progress:  100,
				Timestamp: time.Now(),
			})
		}
	} else {
		if pipeline.Config.NotifyOnError {
			s.sendNotification(Notification{
				Type:      "error",
				TaskID:    run.ID,
				Message:   fmt.Sprintf("Pipeline %s failed: %s", pipeline.Config.Name, lastErr.Error()),
				Progress:  run.Progress,
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"error": lastErr.Error(),
				},
			})
		}
	}

	// 更新流水线状态
	s.mu.Lock()
	pipeline.Status = run.Status
	pipeline.LastRunTime = &now
	s.mu.Unlock()
}

// RegisterPipeline 注册流水线
func (s *Scheduler) RegisterPipeline(pipeline *Pipeline) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.pipelines[pipeline.ID]; exists {
		return fmt.Errorf("pipeline %s already exists", pipeline.ID)
	}

	// 计算下次运行时间
	if pipeline.Config.Schedule != "" {
		next := s.calculateNextRun(pipeline.Config.Schedule)
		pipeline.NextRunTime = &next
	}

	s.pipelines[pipeline.ID] = pipeline
	return nil
}

// UnregisterPipeline 注销流水线
func (s *Scheduler) UnregisterPipeline(id string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.pipelines, id)
}

// TriggerPipeline 手动触发流水线
func (s *Scheduler) TriggerPipeline(id string, triggeredBy string) error {
	s.mu.RLock()
	pipeline, exists := s.pipelines[id]
	s.mu.RUnlock()

	if !exists {
		return fmt.Errorf("pipeline %s not found", id)
	}

	go s.executePipeline(pipeline, "manual", triggeredBy)
	return nil
}

// SendEvent 发送事件
func (s *Scheduler) SendEvent(event Event) {
	select {
	case s.eventQueue <- event:
	default:
		// 队列已满
	}
}

// GetNotificationChannel 获取通知通道
func (s *Scheduler) GetNotificationChannel() <-chan Notification {
	return s.notifyChan
}

// sendNotification 发送通知
func (s *Scheduler) sendNotification(notif Notification) {
	select {
	case s.notifyChan <- notif:
	default:
		// 通知队列已满
	}
}

// calculateNextRun 计算下次运行时间（简化实现）
func (s *Scheduler) calculateNextRun(cronExpr string) time.Time {
	// 简化实现，实际应使用 cron 库解析表达式
	// 这里假设每小时运行一次
	return time.Now().Add(1 * time.Hour)
}

// GetPipelineRun 获取流水线运行记录
func (s *Scheduler) GetPipelineRun(id string) (*models.PipelineRun, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, exists := s.runs[id]
	return run, exists
}

// GetPipelineRuns 获取流水线运行历史
func (s *Scheduler) GetPipelineRuns(pipelineID string, limit int) []*models.PipelineRun {
	s.mu.RLock()
	defer s.mu.RUnlock()

	runs := make([]*models.PipelineRun, 0)
	count := 0

	for _, run := range s.runs {
		if pipelineID == "" || run.PipelineID == pipelineID {
			runs = append(runs, run)
			count++
			if limit > 0 && count >= limit {
				break
			}
		}
	}

	return runs
}

// generateRunID 生成运行ID
func generateRunID() string {
	return fmt.Sprintf("run_%d", time.Now().UnixNano())
}

// EventTrigger 事件触发器
type EventTrigger struct {
	eventType string
}

// NewEventTrigger 创建事件触发器
func NewEventTrigger(eventType string) *EventTrigger {
	return &EventTrigger{eventType: eventType}
}

// ShouldTrigger 检查是否触发
func (t *EventTrigger) ShouldTrigger(event Event) bool {
	return event.Type == t.eventType
}

// GetType 获取触发器类型
func (t *EventTrigger) GetType() string {
	return "event"
}

// TimeTrigger 时间触发器
type TimeTrigger struct {
	interval time.Duration
	lastRun  time.Time
}

// NewTimeTrigger 创建时间触发器
func NewTimeTrigger(interval time.Duration) *TimeTrigger {
	return &TimeTrigger{interval: interval}
}

// ShouldTrigger 检查是否触发
func (t *TimeTrigger) ShouldTrigger(event Event) bool {
	return time.Since(t.lastRun) >= t.interval
}

// GetType 获取触发器类型
func (t *TimeTrigger) GetType() string {
	return "time"
}
