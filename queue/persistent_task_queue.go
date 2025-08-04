package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// PersistentTaskQueue 持久化任务队列实现
type PersistentTaskQueue struct {
	tasks     map[string]*Task
	mutex     sync.RWMutex
	dataFile  string
	dataDir   string
}

// NewPersistentTaskQueue 创建新的持久化任务队列
func NewPersistentTaskQueue(dataDir string) (*PersistentTaskQueue, error) {
	// 确保数据目录存在
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create data directory: %v", err)
	}

	q := &PersistentTaskQueue{
		tasks:    make(map[string]*Task),
		dataDir:  dataDir,
		dataFile: filepath.Join(dataDir, "tasks.json"),
	}

	// 尝试从文件加载现有任务
	if err := q.loadTasks(); err != nil {
		// 如果加载失败，记录警告但不中断初始化
		fmt.Printf("Warning: failed to load tasks from file: %v\n", err)
	}

	return q, nil
}

// loadTasks 从文件加载任务
func (q *PersistentTaskQueue) loadTasks() error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// 检查数据文件是否存在
	if _, err := os.Stat(q.dataFile); os.IsNotExist(err) {
		// 文件不存在，这是正常的初始化情况
		return nil
	}

	// 读取文件内容
	data, err := os.ReadFile(q.dataFile)
	if err != nil {
		return fmt.Errorf("failed to read tasks file: %v", err)
	}

	// 如果文件为空，直接返回
	if len(data) == 0 {
		return nil
	}

	// 解析JSON数据
	var tasks []*Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		return fmt.Errorf("failed to parse tasks data: %v", err)
	}

	// 将任务加载到内存中
	for _, task := range tasks {
		q.tasks[task.ID] = task
	}

	fmt.Printf("Loaded %d tasks from file\n", len(tasks))
	return nil
}

// saveTasks 将任务保存到文件
func (q *PersistentTaskQueue) saveTasks() error {
	// 创建任务列表
	tasks := make([]*Task, 0, len(q.tasks))
	for _, task := range q.tasks {
		tasks = append(tasks, task)
	}

	// 序列化为JSON
	data, err := json.MarshalIndent(tasks, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize tasks: %v", err)
	}

	// 写入文件
	if err := os.WriteFile(q.dataFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write tasks file: %v", err)
	}

	return nil
}

// Push 添加任务到队列
func (q *PersistentTaskQueue) Push(task *Task) error {
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
	
	// 初始化执行次数（如果是新任务）
	if task.ExecutionCount == 0 && task.LastExecution.IsZero() {
		task.ExecutionCount = 1
		task.LastExecution = time.Now()
	}

	q.tasks[task.ID] = task

	// 保存到文件
	return q.saveTasks()
}

// Pop 从队列中取出任务
func (q *PersistentTaskQueue) Pop() (*Task, error) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	// 优先级调度：先查找高优先级任务
	for priority := PriorityCritical; priority >= PriorityLow; priority-- {
		for _, task := range q.tasks {
			if task.Status == "pending" && task.Priority == priority {
				// 更新任务状态为处理中
				task.Status = "processing"
				task.Started = time.Now()
				// 增加执行次数（除了第一次执行）
				if task.ExecutionCount > 0 {
					task.ExecutionCount++
				} else {
					task.ExecutionCount = 1
				}
				task.LastExecution = time.Now()
				
				// 保存到文件
				if err := q.saveTasks(); err != nil {
					return nil, err
				}
				
				return task, nil
			}
		}
	}

	// 如果没有找到按优先级的任务，返回任意pending任务
	for _, task := range q.tasks {
		if task.Status == "pending" {
			// 更新任务状态为处理中
			task.Status = "processing"
			task.Started = time.Now()
			// 增加执行次数（除了第一次执行）
			if task.ExecutionCount > 0 {
				task.ExecutionCount++
			} else {
				task.ExecutionCount = 1
			}
			task.LastExecution = time.Now()
			
			// 保存到文件
			if err := q.saveTasks(); err != nil {
				return nil, err
			}
			
			return task, nil
		}
	}

	return nil, nil
}

// List 列出所有任务
func (q *PersistentTaskQueue) List() ([]*Task, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	tasks := make([]*Task, 0, len(q.tasks))
	for _, task := range q.tasks {
		tasks = append(tasks, task)
	}

	return tasks, nil
}

// Get 获取指定ID的任务
func (q *PersistentTaskQueue) Get(id string) (*Task, error) {
	q.mutex.RLock()
	defer q.mutex.RUnlock()

	task, exists := q.tasks[id]
	if !exists {
		return nil, nil
	}

	return task, nil
}

// Update 更新任务
func (q *PersistentTaskQueue) Update(task *Task) error {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	q.tasks[task.ID] = task

	// 保存到文件
	return q.saveTasks()
}