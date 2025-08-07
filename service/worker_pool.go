package service

import (
	"context"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/utils"
	"github.com/google/uuid"
)

var (
	taskBeingProcessed = make(map[string]bool)
	taskBeingProcessedMutex = sync.RWMutex{}
	maxRetryCount = 3
)

// WorkerPool 工作池结构体
type WorkerPool struct {
	workers     []*Worker
	maxWorkers  int
	taskQueue   queue.TaskQueue
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *utils.Logger
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(maxWorkers int, taskQueue queue.TaskQueue) *WorkerPool {
	// 如果未指定最大工作线程数，则使用CPU核心数
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// 创建日志记录器
	logger, err := utils.NewLogger("./log", fmt.Sprintf("worker_pool_%d", maxWorkers), utils.INFO, 10*1024*1024, 5)
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}
	
	logger.Info("创建工作池", map[string]string{
		"maxWorkers": fmt.Sprintf("%d", maxWorkers),
		"cpuCount":   fmt.Sprintf("%d", runtime.NumCPU()),
	})
	
	return &WorkerPool{
		workers:    make([]*Worker, 0),
		maxWorkers: maxWorkers,
		taskQueue:  taskQueue,
		ctx:        ctx,
		cancel:     cancel,
		logger:     logger,
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	wp.logger.Info("启动工作池", map[string]string{
		"maxWorkers": fmt.Sprintf("%d", wp.maxWorkers),
	})
	
	// 创建并启动工作者
	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(i, wp.taskQueue, wp.ctx)
		wp.workers = append(wp.workers, worker)
		go worker.Start()
		wp.logger.Info("工作者已启动", map[string]string{
			"workerId": fmt.Sprintf("%d", i),
		})
	}
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	wp.logger.Info("停止工作池", nil)
	
	// 取消上下文，通知所有工作者停止
	wp.cancel()
	
	// 等待所有工作者完成
	var wg sync.WaitGroup
	for _, worker := range wp.workers {
		wg.Add(1)
		go func(w *Worker) {
			defer wg.Done()
			w.Stop()
		}(worker)
	}
	wg.Wait()
	
	wp.logger.Info("工作池已停止", nil)
}

// GetWorkerCount 获取工作者数量
func (wp *WorkerPool) GetWorkerCount() int {
	return len(wp.workers)
}

// GetActiveWorkerCount 获取活跃工作者数量
func (wp *WorkerPool) GetActiveWorkerCount() int {
	count := 0
	for _, worker := range wp.workers {
		if worker.IsActive() {
			count++
		}
	}
	return count
}

// Worker 工作者结构体
type Worker struct {
	id       int
	taskQueue queue.TaskQueue
	ctx      context.Context
	cancel   context.CancelFunc
	active   bool
	mutex    sync.Mutex
	logger   *utils.Logger
}

// NewWorker 创建新的工作者
func NewWorker(id int, taskQueue queue.TaskQueue, parentCtx context.Context) *Worker {
	ctx, cancel := context.WithCancel(parentCtx)
	
	// 创建日志记录器
	logger, err := utils.NewLogger("./log", fmt.Sprintf("worker_%d", id), utils.INFO, 10*1024*1024, 5)
	if err != nil {
		log.Fatalf("Failed to create logger for worker %d: %v", id, err)
	}
	
	return &Worker{
		id:        id,
		taskQueue: taskQueue,
		ctx:       ctx,
		cancel:    cancel,
		active:    false,
		logger:    logger,
	}
}

// Start 启动工作者
func (w *Worker) Start() {
	w.logger.Info("工作者开始运行", nil)
	
	for {
		select {
		case <-w.ctx.Done():
			w.logger.Info("工作者收到停止信号", nil)
			return
		default:
			// 尝试从队列中获取任务
			task, err := w.taskQueue.Pop()
			if err != nil {
				if err != queue.ErrEmptyQueue {
					w.logger.Error("从队列获取任务失败", map[string]string{
						"error": err.Error(),
					})
				}
				// 如果队列为空或出错，短暂休眠后重试
				time.Sleep(100 * time.Millisecond)
				continue
			}
			
			// 处理任务
			w.processTask(task)
		}
	}
}

// Stop 停止工作者
func (w *Worker) Stop() {
	w.logger.Info("工作者停止", nil)
	w.cancel()
}

// IsActive 检查工作者是否活跃
func (w *Worker) IsActive() bool {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	return w.active
}

// setActive 设置工作者活跃状态
func (w *Worker) setActive(active bool) {
	w.mutex.Lock()
	defer w.mutex.Unlock()
	w.active = active
}

// processTask 处理任务
func (w *Worker) processTask(task *queue.Task) {
    // 标记工作者为活跃状态
    w.setActive(true)
    defer w.setActive(false)
    defer utils.HandlePanic()
    
    w.logger.Info("开始处理任务", map[string]string{
        "taskId": task.ID,
        "status": task.Status,
    })
    
    // 更新任务状态为处理中
    task.Status = "processing"
    task.Started = time.Now()
    task.Progress = 0.0
    
    // 更新任务到队列，记录执行历史
    if err := w.taskQueue.Update(task); err != nil {
        w.logger.Error("更新任务状态失败", map[string]string{
            "taskId": task.ID,
            "error":  err.Error(),
        })
        return
    }
    
    // 执行任务处理
    result, err := w.executeTask(task)
	
	// 更新任务完成状态
	task.Finished = time.Now()
	
	if err != nil {
		w.logger.Error("任务处理失败", map[string]string{
			"taskId": task.ID,
			"error":  err.Error(),
		})
		
		task.Status = "failed"
		task.Error = err.Error()
		task.Progress = 0.0
	} else {
		w.logger.Info("任务处理完成", map[string]string{
			"taskId": task.ID,
			"result": result,
		})
		
		task.Status = "completed"
		task.Result = result
		task.Progress = 1.0
	}
	
	// 更新任务到队列，记录执行历史
	if err := w.taskQueue.Update(task); err != nil {
		w.logger.Error("更新任务状态失败", map[string]string{
			"taskId": task.ID,
			"error":  err.Error(),
		})
	}
}

// executeTask 执行具体任务
func (w *Worker) executeTask(task *queue.Task) (string, error) {
    // 获取任务类型
    taskType, ok := task.Spec.(map[string]interface{})["taskType"].(string)
    if !ok {
        return "", fmt.Errorf("invalid task type")
    }

    // 如果任务ID为空，则生成新的任务ID
    if task.ID == "" {
        taskID := fmt.Sprintf("m-%s", uuid.New().String())
        task.ID = taskID
    }

    w.logger.Info("执行任务", map[string]string{
        "taskId":   task.ID,
        "taskType": taskType,
    })

    switch taskType {
    case "materialPreprocess":
        return w.executeMaterialPreprocess(task)
    case "videoEdit":
        return w.executeVideoEdit(task)
    default:
        return "", fmt.Errorf("unsupported task type: %s", taskType)
    }
}

// executeMaterialPreprocess 执行素材预处理任务
func (w *Worker) executeMaterialPreprocess(task *queue.Task) (string, error) {
	// 使用MaterialPreprocessorService处理任务，以启用日志功能
	materialPreprocessor := NewMaterialPreprocessorService()
	err := materialPreprocessor.Process(task)
	if err != nil {
		return "", err
	}
	
	// 返回任务结果
	return task.Result, nil
}

// executeVideoEdit 执行视频编辑任务
func (w *Worker) executeVideoEdit(task *queue.Task) (string, error) {
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		return "", fmt.Errorf("invalid task spec")
	}
	
	// 创建临时目录
	tempDir := "./temp"
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		if err := os.MkdirAll(tempDir, 0755); err != nil {
			return "", fmt.Errorf("failed to create temp directory: %v", err)
		}
	}
	
	// 生成输出文件路径
	taskID := task.ID
	outputFile := filepath.Join(tempDir, fmt.Sprintf("%s_output.mp4", taskID))
	
	// 构建FFmpeg命令参数
	args := []string{"-y"} // 覆盖输出文件
	
	// 添加输入文件
	if inputs, ok := spec["inputs"].([]interface{}); ok {
		for _, input := range inputs {
			if inputStr, ok := input.(string); ok {
				args = append(args, "-i", inputStr)
			}
		}
	}
	
	// 添加视频过滤器
	if videoFilter, ok := spec["videoFilter"].(string); ok && videoFilter != "" {
		args = append(args, "-vf", videoFilter)
	}
	
	// 添加其他参数
	if extraArgs, ok := spec["extraArgs"].([]interface{}); ok {
		for _, arg := range extraArgs {
			if argStr, ok := arg.(string); ok {
				args = append(args, argStr)
			}
		}
	}
	
	// 添加输出文件
	args = append(args, outputFile)
	
	w.logger.Info("执行FFmpeg命令", map[string]string{
		"args": strings.Join(args, " "),
	})
	
	// 执行FFmpeg命令
	cmd := exec.CommandContext(w.ctx, "ffmpeg", args...)
	
	// 捕获标准输出和错误输出
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stdout pipe: %v", err)
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return "", fmt.Errorf("failed to get stderr pipe: %v", err)
	}
	
	// 启动命令
	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start ffmpeg: %v", err)
	}
	
	// 实时读取输出
	go func() {
		io.Copy(os.Stdout, stdout)
	}()
	
	go func() {
		io.Copy(os.Stderr, stderr)
	}()
	
	// 等待命令完成
	if err := cmd.Wait(); err != nil {
		return "", fmt.Errorf("ffmpeg execution failed: %v", err)
	}
	
	w.logger.Info("FFmpeg命令执行完成", map[string]string{
		"outputFile": outputFile,
	})
	
	return outputFile, nil
}

// markTaskAsBeingProcessed 标记任务正在处理
func markTaskAsBeingProcessed(taskID string, processing bool) {
	taskBeingProcessedMutex.Lock()
	defer taskBeingProcessedMutex.Unlock()
	
	if processing {
		taskBeingProcessed[taskID] = true
	} else {
		delete(taskBeingProcessed, taskID)
	}
}

// IsTaskBeingProcessed 检查任务是否正在处理
func IsTaskBeingProcessed(taskID string) bool {
	taskBeingProcessedMutex.RLock()
	defer taskBeingProcessedMutex.RUnlock()
	
	_, exists := taskBeingProcessed[taskID]
	return exists
}
