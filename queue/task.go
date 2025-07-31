package queue

import (
	"time"
)

// Task 任务结构体
type Task struct {
	ID        string      `json:"id"`
	Spec      interface{} `json:"spec"`
	Status    string      `json:"status"`
	Created   time.Time   `json:"created"`
	Started   time.Time   `json:"started"`
	Finished  time.Time   `json:"finished"`
	Result    string      `json:"result"`
	Error     string      `json:"error"`
	Progress  float64     `json:"progress"`
}

// TaskQueue 任务队列接口
type TaskQueue interface {
	Add(task *Task) error
	Get(taskID string) (*Task, error)
	List() ([]*Task, error)
	Remove(taskID string) error
	Update(task *Task) error
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