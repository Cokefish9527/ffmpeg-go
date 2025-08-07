package queue

import (
	"fmt"
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

// TaskExecution 任务执行历史记录
type TaskExecution struct {
    ID              string        `json:"id"`
    TaskID          string        `json:"taskId"`
    Status          string        `json:"status"`
    Spec            interface{}   `json:"spec"`
    Result          string        `json:"result"`
    Error           string        `json:"error"`
    Created         time.Time     `json:"created"`
    Started         time.Time     `json:"started"`
    Finished        time.Time     `json:"finished"`
    ExecutionTime   float64       `json:"executionTime"` // 添加 ExecutionTime 字段
    Progress        float64       `json:"progress"`
    Priority        TaskPriority  `json:"priority"`
    ExecutionNumber int           `json:"executionNumber"` // 执行序号
}

// Task 任务结构
type Task struct {
	ID            string        `json:"id"`
	Status        string        `json:"status"`
	Spec          interface{}   `json:"spec"`
	Result        string        `json:"result"`
	Error         string        `json:"error"`
	Created       time.Time     `json:"created"`
	Started       time.Time     `json:"started"`
	Finished      time.Time     `json:"finished"`
	Progress      float64       `json:"progress"`
	Priority      TaskPriority  `json:"priority"` // 添加优先级字段
	ExecutionCount int          `json:"executionCount"` // 添加执行次数字段
	LastExecution  time.Time    `json:"lastExecution"`  // 添加最后执行时间字段
}

// TaskQueue 任务队列接口
type TaskQueue interface {
	Push(task *Task) error
	Pop() (*Task, error)
	List() ([]*Task, error)
	Get(id string) (*Task, error)
	Update(task *Task) error
	GetTaskExecutions(taskID string) ([]*TaskExecution, error) // 添加获取任务执行历史的方法
}

// InMemoryTaskQueue 内存任务队列实现
type InMemoryTaskQueue struct {
	tasks      map[string]*Task
	executions map[string][]*TaskExecution // 任务执行历史记录
	mutex      sync.RWMutex
}

// NewInMemoryTaskQueue 创建新的内存任务队列
func NewInMemoryTaskQueue() *InMemoryTaskQueue {
	return &InMemoryTaskQueue{
		tasks:      make(map[string]*Task),
		executions: make(map[string][]*TaskExecution),
	}
}

// Push 将任务推入队列
func (tq *InMemoryTaskQueue) Push(task *Task) error {
    tq.mutex.Lock()
    defer tq.mutex.Unlock()
    
    // 如果任务已经存在且状态为 pending，则不重复添加
    if existingTask, exists := tq.tasks[task.ID]; exists && existingTask.Status == "pending" {
        return fmt.Errorf("task with ID %s already exists in pending state", task.ID)
    }

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
    
    // 初始化执行次数（如果是新任务）
    if task.ExecutionCount == 0 && task.LastExecution.IsZero() {
        task.ExecutionCount = 1
        task.LastExecution = time.Now()
    }
    
    // 创建初始执行历史记录
    execution := &TaskExecution{
        ID:              uuid.New().String(),
        TaskID:          task.ID,
        Status:          task.Status,
        Spec:            task.Spec,
        Result:          task.Result,
        Error:           task.Error,
        Created:         task.Created,
        Started:         time.Time{},
        Finished:        time.Time{},
        Progress:        task.Progress,
        Priority:        task.Priority,
        ExecutionNumber: task.ExecutionCount,
    }
    
    if tq.executions[task.ID] == nil {
        tq.executions[task.ID] = make([]*TaskExecution, 0)
    }
    tq.executions[task.ID] = append(tq.executions[task.ID], execution)
    
    tq.tasks[task.ID] = task
    return nil
}

// Pop 从队列中取出任务
func (tq *InMemoryTaskQueue) Pop() (*Task, error) {
    task, err := tq.internalPop()
    if err != nil {
        return nil, err
    }

    // 不增加执行次数
    // task.ExecutionCount++

    return task, nil
}

// internalPop 从队列中取出任务
func (tq *InMemoryTaskQueue) internalPop() (*Task, error) {
    tq.mutex.Lock()
    defer tq.mutex.Unlock()
    
    // 优先级调度：先查找高优先级任务
    for priority := PriorityCritical; priority >= PriorityLow; priority-- {
        for _, task := range tq.tasks {
            if task.Status == "pending" && task.Priority == priority {
                // 更新任务状态为处理中并创建执行记录
                return tq.processTask(task)
            }
        }
    }
    
    // 如果没有找到按优先级的任务，返回任意pending任务
    for _, task := range tq.tasks {
        if task.Status == "pending" {
            // 更新任务状态为处理中并创建执行记录
            return tq.processTask(task)
        }
    }
    
    return nil, nil
}

// processTask 处理任务状态更新和执行记录创建
func (tq *InMemoryTaskQueue) processTask(task *Task) (*Task, error) {
    // 更新任务状态为处理中
    task.Status = "processing"
    task.Started = time.Now()
    
    // 不增加执行次数，因为这是任务的第一次执行
    if task.ExecutionCount == 0 {
        task.ExecutionCount = 1
    }
    task.LastExecution = time.Now()

    // 创建执行记录
    execution := &TaskExecution{
        ID:              uuid.New().String(),
        TaskID:          task.ID,
        Status:          task.Status,
        Spec:            task.Spec,
        Result:          task.Result,
        Error:           task.Error,
        Created:         task.Created,
        Started:         task.Started,
        Finished:        time.Time{},
        Progress:        task.Progress,
        Priority:        task.Priority,
        ExecutionNumber: task.ExecutionCount,
    }
    
    if tq.executions[task.ID] == nil {
        tq.executions[task.ID] = make([]*TaskExecution, 0)
    }
    tq.executions[task.ID] = append(tq.executions[task.ID], execution)
    
    return task, nil
}

// GetTaskExecutions 获取任务的所有执行历史
func (tq *InMemoryTaskQueue) GetTaskExecutions(taskID string) ([]*TaskExecution, error) {
	tq.mutex.RLock()
	defer tq.mutex.RUnlock()

	executions, exists := tq.executions[taskID]
	if !exists {
		return nil, nil
	}

	// 返回执行历史的副本
	result := make([]*TaskExecution, len(executions))
	copy(result, executions)
	return result, nil
}

// Update 更新任务状态，并记录执行历史
func (tq *InMemoryTaskQueue) Update(task *Task) error {
    tq.mutex.Lock()
    defer tq.mutex.Unlock()
    
    // 获取当前任务的最新执行历史
    executions, exists := tq.executions[task.ID]
    if !exists || len(executions) == 0 {
        return fmt.Errorf("no execution history found for task %s", task.ID)
    }
    
    latestExecution := executions[len(executions)-1]
    
    // 如果状态发生变化，则创建新的执行历史记录
    if latestExecution.Status != task.Status {
        newExecution := &TaskExecution{
            ID:          uuid.New().String(),
            TaskID:      task.ID,
            Status:      task.Status,
            Spec:        task.Spec,
            Result:      task.Result,
            Error:       task.Error,
            Created:     task.Created,
            Started:     task.Started,
            Finished:    time.Time{}, // 默认为空
            ExecutionTime: 0, // 初始化为0
            Progress:    task.Progress,
            Priority:    task.Priority,
            ExecutionNumber: task.ExecutionCount,
        }
        
        // 如果任务完成或失败，设置结束时间和执行时间
        if task.Status == "completed" || task.Status == "failed" {
            newExecution.Finished = time.Now()
            newExecution.ExecutionTime = newExecution.Finished.Sub(newExecution.Started).Seconds() // 计算并设置执行时间
        }
        
        tq.executions[task.ID] = append(tq.executions[task.ID], newExecution)
        latestExecution = newExecution
    } else {
        // 更新现有执行历史的状态、结束时间和错误信息
        latestExecution.Status = task.Status
        if task.Status == "completed" || task.Status == "failed" {
            latestExecution.Finished = time.Now()
            latestExecution.ExecutionTime = latestExecution.Finished.Sub(latestExecution.Started).Seconds() // 计算并设置执行时间
        }
        latestExecution.Error = task.Error
    }
    
    // 如果任务完成或失败，设置结果
    if task.Status == "completed" || task.Status == "failed" {
        latestExecution.Result = task.Result
    }
    
    // 更新任务到队列
    tq.tasks[task.ID] = task
    return nil
}

// List 列出所有任务
func (tq *InMemoryTaskQueue) List() ([]*Task, error) {
	tq.mutex.RLock()
	defer tq.mutex.RUnlock()
	
	tasks := make([]*Task, 0, len(tq.tasks))
	for _, task := range tq.tasks {
		tasks = append(tasks, task)
	}
	
	return tasks, nil
}

// Get 获取指定ID的任务
func (tq *InMemoryTaskQueue) Get(id string) (*Task, error) {
	tq.mutex.RLock()
	defer tq.mutex.RUnlock()
	
	task, exists := tq.tasks[id]
	if !exists {
		return nil, nil
	}
	
	return task, nil
}
