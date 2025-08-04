package service

import (
	"time"
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

// InMemoryTaskQueue 内存任务队列实现
type InMemoryTaskQueue struct {
	tasks map[string]*Task
}

// NewInMemoryTaskQueue 创建新的内存任务队列实例
func NewInMemoryTaskQueue() *InMemoryTaskQueue {
	return &InMemoryTaskQueue{
		tasks: make(map[string]*Task),
	}
}

// Add 添加任务到队列
func (q *InMemoryTaskQueue) Add(task *Task) error {
	q.tasks[task.ID] = task
	return nil
}

// Get 根据任务ID获取任务
func (q *InMemoryTaskQueue) Get(taskID string) (*Task, error) {
	task, exists := q.tasks[taskID]
	if !exists {
		return nil, nil
	}
	return task, nil
}

// List 获取所有任务列表
func (q *InMemoryTaskQueue) List() ([]*Task, error) {
	tasks := make([]*Task, 0, len(q.tasks))
	for _, task := range q.tasks {
		tasks = append(tasks, task)
	}
	return tasks, nil
}

// Remove 根据任务ID移除任务
func (q *InMemoryTaskQueue) Remove(taskID string) error {
	delete(q.tasks, taskID)
	return nil
}

// Update 更新任务信息
func (q *InMemoryTaskQueue) Update(task *Task) error {
	q.tasks[task.ID] = task
	return nil
}