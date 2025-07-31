package service

import (
	"context"
	"time"
	
	"github.com/u2takey/ffmpeg-go/queue"
)

// WorkerPool 工作池结构，用于控制并发任务数
type WorkerPool struct {
	maxWorkers int           // 最大工作协程数
	taskQueue  chan *queue.Task // 任务队列
	semaphore  chan struct{}    // 信号量，用于限制并发数
	workers    []*Worker        // 工作协程列表
	ctx        context.Context  // 上下文，用于控制工作池生命周期
	cancel     context.CancelFunc // 取消函数，用于停止工作池
}

// Worker 工作协程结构
type Worker struct {
	id     int              // 工作协程ID
	pool   *WorkerPool      // 所属工作池
	ctx    context.Context  // 上下文
	cancel context.CancelFunc // 取消函数
}

// NewWorkerPool 创建新的工作池实例
func NewWorkerPool(maxWorkers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &WorkerPool{
		maxWorkers: maxWorkers,
		taskQueue:  make(chan *queue.Task, 1000), // 任务队列缓冲区大小为1000
		semaphore:  make(chan struct{}, maxWorkers), // 信号量大小等于最大工作协程数
		ctx:        ctx,
		cancel:     cancel,
	}
	
	// 创建并启动工作协程
	pool.workers = make([]*Worker, maxWorkers)
	for i := 0; i < maxWorkers; i++ {
		workerCtx, workerCancel := context.WithCancel(ctx)
		worker := &Worker{
			id:     i,
			pool:   pool,
			ctx:    workerCtx,
			cancel: workerCancel,
		}
		pool.workers[i] = worker
		go worker.Start()
	}
	
	return pool
}

// SubmitTask 提交任务到工作池
func (wp *WorkerPool) SubmitTask(task *queue.Task) error {
	// 检查上下文是否已取消
	select {
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	default:
	}
	
	// 将任务发送到任务队列
	select {
	case wp.taskQueue <- task:
		return nil
	case <-wp.ctx.Done():
		return wp.ctx.Err()
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	// 工作协程已经在NewWorkerPool中启动
	// 这里可以添加其他启动逻辑
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	// 取消上下文，通知所有工作协程停止
	wp.cancel()
	
	// 等待一段时间让工作协程优雅退出
	time.Sleep(100 * time.Millisecond)
}

// Start 工作协程开始处理任务
func (w *Worker) Start() {
	for {
		select {
		case <-w.ctx.Done():
			// 收到停止信号，退出循环
			return
		case task := <-w.pool.taskQueue:
			// 获取信号量，限制并发数
			select {
			case w.pool.semaphore <- struct{}{}:
				// 处理任务
				w.processTask(task)
				
				// 释放信号量
				<-w.pool.semaphore
			case <-w.ctx.Done():
				// 收到停止信号，退出循环
				return
			}
		}
	}
}

// processTask 处理单个任务
func (w *Worker) processTask(task *queue.Task) {
	// 更新任务状态为处理中
	task.Status = "processing"
	task.Started = time.Now()

	// 尝试将任务的Spec转换为EditSpec
	spec, ok := task.Spec.(*ffmpeg.EditSpec)
	if !ok {
		// 如果转换失败，尝试通过map进行解析
		if specMap, ok := task.Spec.(map[string]interface{}); ok {
			// 创建一个新的EditSpec实例
			spec = &ffmpeg.EditSpec{}

			// 简化处理，实际项目中应该使用更完善的解析方法
			if outPath, exists := specMap["outPath"]; exists {
				if str, ok := outPath.(string); ok {
					spec.OutPath = str
				}
			}

			if width, exists := specMap["width"]; exists {
				if num, ok := width.(float64); ok {
					spec.Width = int(num)
				}
			}

			if height, exists := specMap["height"]; exists {
				if num, ok := height.(float64); ok {
					spec.Height = int(num)
				}
			}

			if fps, exists := specMap["fps"]; exists {
				if num, ok := fps.(float64); ok {
					spec.Fps = int(num)
				}
			}
		}
	}

	// 如果仍然没有有效的spec，则返回错误
	if spec == nil {
		task.Status = "failed"
		task.Error = "无效的视频编辑规范"
		task.Finished = time.Now()
		return
	}

	// 调用ffmpeg-go库进行视频编辑
	err := ffmpeg.Edit(spec)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
	} else {
		task.Status = "completed"
		task.Result = spec.OutPath
	}

	task.Finished = time.Now()
}