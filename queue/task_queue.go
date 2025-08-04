package queue

import (
	"sync"
	"time"
	
	"github.com/google/uuid"
)

// TaskPriority 任务优先级
type TaskPriority int

const (
	PriorityLow TaskPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Task 任务结构
type Task struct {
	ID        string        `json:"id"`
	Status    string        `json:"status"`
	Spec      interface{}   `json:"spec"`
	Result    string        `json:"result"`
	Error     string        `json:"error"`
	Created   time.Time     `json:"created"`
	Started   time.Time     `json:"started"`
	Finished  time.Time     `json:"finished"`
	Progress  float64       `json:"progress"`
	Priority  TaskPriority  `json:"priority"` // 添加优先级字段
}

// TaskQueue 任务队列接口
type TaskQueue interface {
	Push(task *Task) error
	Pop() (*Task, error)
	List() ([]*Task, error)
	Get(id string) (*Task, error)
	Update(task *Task) error
}

// InMemoryTaskQueue 内存任务队列实现
type InMemoryTaskQueue struct {
	tasks map[string]*Task
	mutex sync.RWMutex
}

// NewInMemoryTaskQueue 创建新的内存任务队列
func NewInMemoryTaskQueue() *InMemoryTaskQueue {
	return &InMemoryTaskQueue{
		tasks: make(map[string]*Task),
	}
}

// Push 添加任务到队列
func (q *InMemoryTaskQueue) Push(task *Task) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	// 如果任务没有ID，则生成一个
	if task.ID == "" {
		task.ID = uuid.New().String()
	}
	
	// 设置默认状态和创建时间
	if task.Status == "" {
		task.Status = "pending"
	}
	
	if task.Created.IsZero() {
		task.Created = time.Now()
	}
	
	// 设置默认优先级
	if task.Priority == 0 {
		task.Priority = PriorityNormal
	}
	
	q.tasks[task.ID] = task
	return nil
}

// Pop 从队列中取出任务
func (q *InMemoryTaskQueue) Pop() (*Task, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	// 优先级调度：先查找高优先级任务
	for priority := PriorityCritical; priority >= PriorityLow; priority-- {
		for _, task := range q.tasks {
			if task.Status == "pending" && task.Priority == priority {
				return task, nil
			}
		}
	}
	
	// 如果没有找到按优先级的任务，返回任意pending任务
	for _, task := range q.tasks {
		if task.Status == "pending" {
			return task, nil
		}
	}
	
	return nil, nil
}

// List 列出所有任务
func (q *InMemoryTaskQueue) List() ([]*Task, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	
	tasks := make([]*Task, 0, len(q.tasks))
	for _, task := range q.tasks {
		tasks = append(tasks, task)
	}
	
	return tasks, nil
}

// Get 获取指定ID的任务
func (q *InMemoryTaskQueue) Get(id string) (*Task, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()
	
	task, exists := q.tasks[id]
	if !exists {
		return nil, nil
	}
	
	return task, nil
}

// Update 更新任务
func (q *InMemoryTaskQueue) Update(task *Task) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()
	
	q.tasks[task.ID] = task
	return nil
}