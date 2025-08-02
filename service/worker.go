package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// WorkerPool 工作池结构，用于控制并发任务数
type WorkerPool struct {
	maxWorkers int         // 最大工作协程数
	taskQueue  chan *Task  // 任务队列
	semaphore  chan struct{}     // 信号量，用于限制并发数
	workers    []*Worker         // 工作协程列表
	ctx        context.Context   // 上下文，用于控制工作池生命周期
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
		taskQueue:  make(chan *Task, 1000), // 任务队列缓冲区大小为1000
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
func (wp *WorkerPool) SubmitTask(task *Task) error {
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
func (w *Worker) processTask(task *Task) {
	// 更新任务状态为处理中
	task.Status = "processing"
	task.Started = time.Now()

	// 尝试将任务的Spec转换为EditSpec
	// 这里暂时简化处理，实际项目中应该使用更完善的解析方法
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		// 如果转换失败，返回错误
		task.Status = "failed"
		task.Error = "无效的视频编辑规范"
		task.Finished = time.Now()
		return
	}

	// 简化处理，实际项目中应该使用更完善的解析方法
	outPath := ""
	if outPathVal, exists := spec["outPath"]; exists {
		if str, ok := outPathVal.(string); ok {
			outPath = str
		}
	}

	width := 0
	if widthVal, exists := spec["width"]; exists {
		if num, ok := widthVal.(float64); ok {
			width = int(num)
		}
	}

	height := 0
	if heightVal, exists := spec["height"]; exists {
		if num, ok := heightVal.(float64); ok {
			height = int(num)
		}
	}

	fps := 0
	if fpsVal, exists := spec["fps"]; exists {
		if num, ok := fpsVal.(float64); ok {
			fps = int(num)
		}
	}

	// 模拟视频编辑处理
	// 实际项目中应该调用ffmpeg-go库进行视频编辑
	// 这里我们模拟实际的视频处理操作
	time.Sleep(2 * time.Second)

	// 实际执行视频合并操作
	if outPath != "" && width > 0 && height > 0 {
		err := w.mergeVideos(outPath, width, height, fps)
		if err != nil {
			task.Status = "failed"
			task.Error = fmt.Sprintf("视频合并失败: %v", err)
		} else {
			task.Status = "completed"
			task.Result = outPath
		}
	} else {
		task.Status = "failed"
		task.Error = "缺少必要的视频编辑参数"
	}

	task.Finished = time.Now()
}

// mergeVideos 合并视频文件
func (w *Worker) mergeVideos(outPath string, width, height, fps int) error {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("无法获取当前工作目录: %v", err)
	}

	// 创建输入文件列表（使用绝对路径）
	inputFiles := []string{
		filepath.Join(wd, "video", "1.mp4"),
		filepath.Join(wd, "video", "2.mp4"),
		filepath.Join(wd, "video", "3.mp4"),
		filepath.Join(wd, "video", "4.mp4"),
	}

	// 检查所有输入文件是否存在
	for _, file := range inputFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			return fmt.Errorf("输入文件不存在: %s", file)
		}
	}

	// 创建临时文件列表用于concat
	listFile := filepath.Join(wd, "video", "file_list.txt")
	file, err := os.Create(listFile)
	if err != nil {
		return fmt.Errorf("无法创建文件列表: %v", err)
	}
	defer os.Remove(listFile)

	for _, input := range inputFiles {
		// 在列表文件中使用双反斜杠转义路径
		fmt.Fprintf(file, "file '%s'\n", strings.ReplaceAll(input, "\\", "/"))
	}
	file.Close()

	// 使用ffmpeg合并视频，强制重新编码以确保兼容性
	// 构建命令: ffmpeg -f concat -safe 0 -i file_list.txt -vf scale=width:height,fps=fps -c:v libx264 -crf 23 -preset medium -c:a aac -b:a 128k outPath
	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFile,
		"-vf", fmt.Sprintf("scale=%d:%d,fps=%d", width, height, fps),
		"-c:v", "libx264", "-crf", "23", "-preset", "medium",
		"-c:a", "aac", "-b:a", "128k",
		outPath, "-y")

	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg执行失败: %v, 输出: %s", err, string(output))
	}

	// 检查输出文件是否存在
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		return fmt.Errorf("输出文件未生成")
	}

	return nil
}
