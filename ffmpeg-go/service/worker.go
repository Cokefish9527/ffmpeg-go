package service

import (
	"log"
	"sync"
	"time"

	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
)

// Worker 工作者接口
type Worker interface {
	Start()
	Stop()
}

// WorkerImpl 工作者实现
type WorkerImpl struct {
	id       int
	taskQueue queue.TaskQueue
	running  bool
	mu       sync.Mutex
}

// NewWorker 创建新的工作者
func NewWorker(id int, taskQueue queue.TaskQueue) Worker {
	return &WorkerImpl{
		id:        id,
		taskQueue: taskQueue,
		running:   false,
	}
}

// Start 启动工作者
func (w *WorkerImpl) Start() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.running {
		return
	}

	w.running = true
	go w.work()
}

// Stop 停止工作者
func (w *WorkerImpl) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.running = false
}

// work 工作者处理任务的主循环
func (w *WorkerImpl) work() {
	for {
		w.mu.Lock()
		if !w.running {
			w.mu.Unlock()
			return
		}
		w.mu.Unlock()

		// 从队列中获取任务
		task, err := w.taskQueue.Pop()
		if err != nil {
			log.Printf("Worker %d: Error popping task from queue: %v", w.id, err)
			continue
		}

		if task == nil {
			// 没有任务，短暂休眠
			time.Sleep(100 * time.Millisecond)
			continue
		}

		// 处理任务
		log.Printf("Worker %d: Processing task %s", w.id, task.ID)
		err = w.processTask(task)
		if err != nil {
			log.Printf("Worker %d: Error processing task %s: %v", w.id, task.ID, err)
			task.Status = "failed"
			task.Error = err.Error()
		} else {
			task.Status = "completed"
			task.Progress = 1.0
		}

		task.Finished = time.Now()

		// 更新任务状态
		err = w.taskQueue.Update(task)
		if err != nil {
			log.Printf("Worker %d: Error updating task %s: %v", w.id, task.ID, err)
		}

		log.Printf("Worker %d: Completed task %s", w.id, task.ID)
	}
}

// processTask 处理单个任务
func (w *WorkerImpl) processTask(task *queue.Task) error {
	// 设置任务开始时间
	task.Started = time.Now()
	
	// 更新任务状态到队列
	err := w.taskQueue.Update(task)
	if err != nil {
		return err
	}

	// 根据任务类型进行不同的处理
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		return w.processDefaultTask(task)
	}

	taskType, ok := spec["taskType"].(string)
	if !ok {
		return w.processDefaultTask(task)
	}

	switch taskType {
	case "materialPreprocess":
		// 处理素材预处理任务
		preprocessor := service.NewMaterialPreprocessorService()
		return preprocessor.Process(task)
	default:
		return w.processDefaultTask(task)
	}
}

// processDefaultTask 处理默认任务（向后兼容）
func (w *WorkerImpl) processDefaultTask(task *queue.Task) error {
	task.Status = "processing"
	
	// 模拟处理过程
	for i := 0; i <= 10; i++ {
		time.Sleep(100 * time.Millisecond) // 模拟处理时间
		task.Progress = float64(i) / 10.0
		
		// 更新进度
		err := w.taskQueue.Update(task)
		if err != nil {
			return err
		}
	}

	return nil
}