package queue

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

// Task 任务结构
type Task struct {
	ID       string      `json:"id"`
	Spec     interface{} `json:"spec"`
	Status   string      `json:"status"`
	Progress float64     `json:"progress"`
	Result   string      `json:"result,omitempty"`
	Error    string      `json:"error,omitempty"`
	Created  time.Time   `json:"created"`
	Started  time.Time   `json:"started,omitempty"`
	Finished time.Time   `json:"finished,omitempty"`
}

// TaskQueue 任务队列接口
type TaskQueue interface {
	Add(task *Task) error
	Get(taskID string) (*Task, error)
	List() ([]*Task, error)
	Update(task *Task) error
	// ProcessNext() (*Task, error) // 处理下一个任务
}

// InMemoryTaskQueue 内存任务队列实现
type InMemoryTaskQueue struct {
	tasks map[string]*Task
	mu    sync.RWMutex
}

// NewInMemoryTaskQueue 创建新的内存任务队列
func NewInMemoryTaskQueue() *InMemoryTaskQueue {
	return &InMemoryTaskQueue{
		tasks: make(map[string]*Task),
	}
}

// Add 添加任务到队列
func (q *InMemoryTaskQueue) Add(task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if task.ID == "" {
		task.ID = uuid.New().String()
	}

	if task.Created.IsZero() {
		task.Created = time.Now()
	}

	task.Status = "pending"
	q.tasks[task.ID] = task
	return nil
}

// Get 根据ID获取任务
func (q *InMemoryTaskQueue) Get(taskID string) (*Task, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	task, exists := q.tasks[taskID]
	if !exists {
		return nil, nil
	}
	return task, nil
}

// List 获取所有任务
func (q *InMemoryTaskQueue) List() ([]*Task, error) {
	q.mu.RLock()
	defer q.mu.RUnlock()

	tasks := make([]*Task, 0, len(q.tasks))
	for _, task := range q.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// Update 更新任务
func (q *InMemoryTaskQueue) Update(task *Task) error {
	q.mu.Lock()
	defer q.mu.Unlock()

	q.tasks[task.ID] = task
	return nil
}