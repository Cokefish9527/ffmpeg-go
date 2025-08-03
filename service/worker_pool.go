package service

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

var (
	taskBeingProcessed = make(map[string]bool)
	taskMutex          = sync.Mutex{}
	// 任务超时时间
	taskTimeout = 30 * time.Minute
)

// WorkerPool 工作池结构
type WorkerPool struct {
	workers   []*Worker
	taskQueue TaskQueue
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(size int, taskQueue TaskQueue) *WorkerPool {
	// 如果未指定大小，则使用CPU核心数
	if size <= 0 {
		size = runtime.NumCPU()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	wp := &WorkerPool{
		workers:   make([]*Worker, size),
		taskQueue: taskQueue,
		ctx:       ctx,
		cancel:    cancel,
	}

	// 创建工作线程
	for i := 0; i < size; i++ {
		worker := NewWorker(i, taskQueue)
		wp.workers[i] = worker
	}

	return wp
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	for _, worker := range wp.workers {
		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.Run(wp.ctx)
		}(worker)
	}
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	wp.cancel()
	wp.wg.Wait()
}

// GetWorkerCount 获取工作线程数
func (wp *WorkerPool) GetWorkerCount() int {
	return len(wp.workers)
}

// SubmitTask 提交任务到工作池
func (wp *WorkerPool) SubmitTask(task *Task) error {
	return wp.taskQueue.Add(task)
}

// Worker 工作线程结构
type Worker struct {
	id        int
	taskQueue TaskQueue
}

// NewWorker 创建新的工作线程
func NewWorker(id int, taskQueue TaskQueue) *Worker {
	return &Worker{
		id:        id,
		taskQueue: taskQueue,
	}
}

// Run 运行工作线程
func (w *Worker) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		default:
			// 处理任务
			w.processNextTask()
			// 短暂休眠以避免过度占用CPU
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// processNextTask 处理下一个任务
func (w *Worker) processNextTask() {
	// 获取所有任务
	tasks, err := w.taskQueue.List()
	if err != nil {
		return
	}

	// 查找待处理的任务
	for _, task := range tasks {
		if task.Status == "pending" {
			// 使用互斥锁确保只有一个Worker处理这个任务
			taskMutex.Lock()
			if !taskBeingProcessed[task.ID] {
				taskBeingProcessed[task.ID] = true
				taskMutex.Unlock()
				
				// 更新任务状态为处理中
				task.Status = "processing"
				task.Started = time.Now()
				w.taskQueue.Update(task)
				
				// 处理任务
				w.processTask(task)
				return
			}
			taskMutex.Unlock()
		}
	}
}

// processTask 处理单个任务
func (w *Worker) processTask(task *Task) {
	// 尝试将任务的Spec转换为EditSpec
	// 这里暂时简化处理，实际项目中应该使用更完善的解析方法
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		// 如果转换失败，返回错误
		task.Status = "failed"
		task.Error = "无效的视频编辑规范"
		task.Finished = time.Now()
		w.taskQueue.Update(task)
		
		// 清理任务处理标记
		taskMutex.Lock()
		delete(taskBeingProcessed, task.ID)
		taskMutex.Unlock()
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

	// 获取输入文件列表（如果存在）
	var inputFiles []string
	if inputVal, exists := spec["inputs"]; exists {
		if inputs, ok := inputVal.([]interface{}); ok {
			for _, input := range inputs {
				if inputStr, ok := input.(string); ok {
					inputFiles = append(inputFiles, inputStr)
				}
			}
		}
	}
	
	// 如果没有提供输入文件列表，则使用默认列表
	if len(inputFiles) == 0 {
		inputFiles = []string{
			"1.mp4",
			"2.mp4",
			"3.mp4",
			"4.mp4",
		}
	}

	// 根据视频质量和目标质量选择合适的编码预设
	preset := "medium" // 默认预设
	
	// 如果目标分辨率较低，可以使用更快的编码
	if width <= 640 && height <= 480 {
		preset = "fast"
	}
	
	// 如果目标分辨率很高，使用较慢但质量更好的编码
	if width >= 1920 && height >= 1080 {
		preset = "slow"
	}

	// 实际执行视频合并操作
	if outPath != "" && width > 0 && height > 0 {
		err := w.mergeVideos(inputFiles, outPath, width, height, fps, preset)
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
	w.taskQueue.Update(task)
	
	// 清理任务处理标记
	taskMutex.Lock()
	delete(taskBeingProcessed, task.ID)
	taskMutex.Unlock()
}

// mergeVideos 合并视频文件
func (w *Worker) mergeVideos(inputFiles []string, outPath string, width, height, fps int, preset string) error {
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("无法获取当前工作目录: %v", err)
	}

	// 创建输入文件列表（使用绝对路径）
	var fullInputFiles []string
	for _, file := range inputFiles {
		fullPath := filepath.Join(wd, "video", file)
		fullInputFiles = append(fullInputFiles, fullPath)
	}

	// 检查所有输入文件是否存在
	for _, file := range fullInputFiles {
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

	for _, input := range fullInputFiles {
		// 在列表文件中使用双反斜杠转义路径
		fmt.Fprintf(file, "file '%s'\n", strings.ReplaceAll(input, "\\", "/"))
	}
	file.Close()

	// 使用ffmpeg合并视频，根据preset选择合适的编码速度
	// 构建命令: ffmpeg -f concat -safe 0 -i file_list.txt -vf scale=width:height,fps=fps -c:v libx264 -crf 23 -preset preset -c:a aac -b:a 128k outPath
	cmd := exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFile,
		"-vf", fmt.Sprintf("scale=%d:%d,fps=%d", width, height, fps),
		"-c:v", "libx264", "-crf", "23", "-preset", preset,
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
