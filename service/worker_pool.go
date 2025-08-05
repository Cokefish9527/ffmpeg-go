package service

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
	
	"github.com/pkg/errors"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/utils"
)

var (
	taskBeingProcessed = make(map[string]bool)
	taskMutex          = sync.Mutex{}
	videoInfoCache     = NewVideoInfoCache()     // 全局视频信息缓存
	processingCache    = GlobalProcessingCache   // 全局处理缓存
	bufferPool         = GlobalBufferPool        // 全局缓冲池
	framePool          = GlobalFramePool         // 全局帧池
)

// WorkerPool 工作池结构
type WorkerPool struct {
	workers    []*Worker
	maxWorkers int
	taskQueue  queue.TaskQueue
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup
	mutex      sync.Mutex
	goroutinePool *utils.GoroutinePool // Goroutine池
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(maxWorkers int, taskQueue queue.TaskQueue) *WorkerPool {
	// 如果未指定最大工作线程数，则使用CPU核心数
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	// 创建Goroutine池
	goroutinePool := utils.NewGoroutinePool(
		utils.WithMinWorkers(int32(maxWorkers/2)),     // 最小工作线程数为配置的一半
		utils.WithMaxWorkers(int32(maxWorkers*2)),     // 最大工作线程数为配置的两倍
		utils.WithTaskQueueSize(10000),                // 任务队列大小
		utils.WithWorkerTimeout(time.Minute*5),        // 工作线程超时时间
		utils.WithTaskTimeout(time.Hour),              // 任务超时时间（视频处理可能较长）
	)
	
	utils.Info("创建工作池", map[string]string{
		"maxWorkers": fmt.Sprintf("%d", maxWorkers),
		"cpuCount":   fmt.Sprintf("%d", runtime.NumCPU()),
	})
	
	return &WorkerPool{
		workers:       make([]*Worker, 0),
		maxWorkers:    maxWorkers,
		taskQueue:     taskQueue,
		ctx:           ctx,
		cancel:        cancel,
		goroutinePool: goroutinePool,
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	// 启动Goroutine池
	wp.goroutinePool.Start()
	
	// 初始化并启动工作线程
	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(wp.taskQueue, wp.goroutinePool)
		wp.workers = append(wp.workers, worker)
		
		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.Start(wp.ctx)
		}(worker)
	}
	
	utils.Info("工作池启动完成", map[string]string{"workerCount": fmt.Sprintf("%d", wp.maxWorkers)})
	fmt.Printf("WorkerPool started with %d workers\n", wp.maxWorkers)
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	utils.Info("正在停止工作池", nil)
	
	// 取消上下文，通知所有工作线程停止
	wp.cancel()
	
	// 等待所有工作线程完成
	wp.wg.Wait()
	
	// 停止Goroutine池
	wp.goroutinePool.Stop()
	
	utils.Info("工作池已停止", nil)
	fmt.Println("WorkerPool stopped")
}

// Resize 调整工作池大小
func (wp *WorkerPool) Resize(newSize int) {
	if newSize <= 0 {
		newSize = runtime.NumCPU()
	}
	
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	currentSize := len(wp.workers)
	
	utils.Info("调整工作池大小", map[string]string{
		"currentSize": fmt.Sprintf("%d", currentSize),
		"newSize":     fmt.Sprintf("%d", newSize),
	})
	
	if newSize > currentSize {
		// 增加Worker数量
		for i := currentSize; i < newSize; i++ {
			worker := NewWorker(wp.taskQueue, wp.goroutinePool)
			wp.workers = append(wp.workers, worker)
			
			wp.wg.Add(1)
			go func(w *Worker) {
				defer wp.wg.Done()
				w.Start(wp.ctx)
			}(worker)
		}
		utils.Info("工作池扩容完成", map[string]string{
			"addedWorkers": fmt.Sprintf("%d", newSize-currentSize),
			"totalWorkers": fmt.Sprintf("%d", newSize),
		})
		fmt.Printf("WorkerPool resized from %d to %d workers (added %d workers)\n", currentSize, newSize, newSize-currentSize)
	} else if newSize < currentSize {
		// 减少Worker数量（这里简化处理，实际项目中应该更优雅地处理）
		wp.workers = wp.workers[:newSize]
		utils.Info("工作池缩容完成", map[string]string{
			"removedWorkers": fmt.Sprintf("%d", currentSize-newSize),
			"totalWorkers":   fmt.Sprintf("%d", newSize),
		})
		fmt.Printf("WorkerPool resized from %d to %d workers (removed %d workers)\n", currentSize, newSize, currentSize-newSize)
		// 注意：实际项目中需要更仔细地处理正在运行的Worker
	}
}

// GetWorkerCount 获取当前Worker数量
func (wp *WorkerPool) GetWorkerCount() int {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	return len(wp.workers)
}

// GetActiveWorkerCount 获取活跃Worker数量
func (wp *WorkerPool) GetActiveWorkerCount() int {
    wp.mutex.Lock()
    defer wp.mutex.Unlock()
    
    activeCount := 0
    for _, worker := range wp.workers {
        if worker.IsActive() { // 使用IsActive方法来判断是否活跃
            activeCount++
        }
    }
    return activeCount
}

// 检测可用的硬件编码器
func detectHardwareEncoders() map[string]bool {
	utils.Debug("检测硬件编码器", nil)
	
	encoders := make(map[string]bool)
	
	// 检测NVIDIA NVENC
	cmd := exec.Command("ffmpeg", "-h", "encoder=h264_nvenc")
	if err := cmd.Run(); err == nil {
		encoders["h264_nvenc"] = true
		utils.Debug("检测到NVIDIA NVENC编码器", nil)
	}
	
	// 检测Intel Quick Sync
	cmd = exec.Command("ffmpeg", "-h", "encoder=h264_qsv")
	if err := cmd.Run(); err == nil {
		encoders["h264_qsv"] = true
		utils.Debug("检测到Intel Quick Sync编码器", nil)
	}
	
	// 检测AMD VCE
	cmd = exec.Command("ffmpeg", "-h", "encoder=h264_amf")
	if err := cmd.Run(); err == nil {
		encoders["h264_amf"] = true
		utils.Debug("检测到AMD VCE编码器", nil)
	}
	
	utils.Debug("硬件编码器检测完成", map[string]string{
		"nvenc": fmt.Sprintf("%t", encoders["h264_nvenc"]),
		"qsv":   fmt.Sprintf("%t", encoders["h264_qsv"]),
		"amf":   fmt.Sprintf("%t", encoders["h264_amf"]),
	})
	
	return encoders
}

// 选择最佳编码器
func selectBestEncoder() string {
	availableEncoders := detectHardwareEncoders()
	
	// 优先级顺序：NVENC > QSV > AMF > libx264
	if availableEncoders["h264_nvenc"] {
		utils.Info("选择NVIDIA NVENC作为编码器", nil)
		return "h264_nvenc"
	}
	
	if availableEncoders["h264_qsv"] {
		utils.Info("选择Intel Quick Sync作为编码器", nil)
		return "h264_qsv"
	}
	
	if availableEncoders["h264_amf"] {
		utils.Info("选择AMD VCE作为编码器", nil)
		return "h264_amf"
	}
	
	// 默认使用libx264
	utils.Info("选择libx264作为编码器", nil)
	return "libx264"
}

// 尝试使用指定编码器，如果失败则降级到libx264
func tryEncoderWithFallback(encoder, listFile, outPath string, width, height, fps int, preset string) error {
	utils.Info("尝试使用编码器", map[string]string{
		"encoder": encoder,
		"width":   fmt.Sprintf("%d", width),
		"height":  fmt.Sprintf("%d", height),
		"fps":     fmt.Sprintf("%d", fps),
		"preset":  preset,
	})
	
	// 首先尝试使用指定的编码器
	err := runFFmpegWithEncoder(encoder, listFile, outPath, width, height, fps, preset)
	if err == nil {
		utils.Info("编码器使用成功", map[string]string{"encoder": encoder})
		return nil // 成功则直接返回
	}
	
	// 如果失败且不是libx264，则尝试降级到libx264
	if encoder != "libx264" {
		utils.Warn("编码器使用失败，尝试降级到libx264", map[string]string{
			"encoder": encoder,
			"error":   err.Error(),
		})
		fmt.Printf("使用编码器 %s 失败，降级到 libx264: %v\n", encoder, err)
		return runFFmpegWithEncoder("libx264", listFile, outPath, width, height, fps, preset)
	}
	
	// 如果已经是libx264还失败，则返回错误
	utils.Error("libx264编码器使用失败", map[string]string{"error": err.Error()})
	return err
}

// Worker 工作者结构
type Worker struct {
    id            int
    taskQueue     queue.TaskQueue
    goroutinePool *utils.GoroutinePool
    isActive      bool // 添加isActive字段表示工作状态
}

// IsActive 检查工作线程是否活跃
func (w *Worker) IsActive() bool {
    return w.isActive
}

// NewWorker 创建新的工作者
func NewWorker(taskQueue queue.TaskQueue, goroutinePool *utils.GoroutinePool) *Worker {
    utils.Debug("创建新的工作者", nil)
    
    return &Worker{
        id:            0, // 实际项目中应该分配唯一ID
        taskQueue:     taskQueue,
        goroutinePool: goroutinePool,
        isActive:      true, // 初始化为活跃状态
    }
}

// Start 启动工作者
func (w *Worker) Start(ctx context.Context) {
    utils.Info("工作者启动", nil)
    
    // 设置 isActive 为 true
    w.isActive = true // 确保在启动时设置为 true

    // 持续处理任务直到上下文被取消
    for {
        select {
        case <-ctx.Done():
            // 上下文被取消，退出循环
            utils.Info("工作者收到停止信号", nil)
            w.isActive = false // 在停止时设置为 false
            return
        default:
            // 处理下一个任务
            w.processNextTask()
            
            // 短暂休眠以避免过度占用CPU
            time.Sleep(100 * time.Millisecond)
        }
    }
}

// processNextTask 处理下一个任务
func (w *Worker) processNextTask() {
	// 获取所有任务
	tasks, err := w.taskQueue.List()
	if err != nil {
		utils.Error("获取任务列表失败", map[string]string{"error": err.Error()})
		fmt.Printf("获取任务列表失败: %v\n", err)
		return
	}

	// 查找待处理的任务（按优先级顺序）
	for priority := queue.PriorityCritical; priority >= queue.PriorityLow; priority-- {
		for _, task := range tasks {
			if task.Status == "pending" && task.Priority == priority {
				// 使用互斥锁确保只有一个Worker处理这个任务
				taskMutex.Lock()
				if !taskBeingProcessed[task.ID] {
					taskBeingProcessed[task.ID] = true
					taskMutex.Unlock()
					
					utils.Info("开始处理任务", map[string]string{
						"taskId":   task.ID,
						"priority": fmt.Sprintf("%d", task.Priority),
						"executionCount": fmt.Sprintf("%d", task.ExecutionCount),
					})
					
					// 更新任务状态为处理中
					task.Status = "processing"
					task.Started = time.Now()
					// 更新执行次数和最后执行时间
					task.ExecutionCount++
					task.LastExecution = time.Now()
					err := w.taskQueue.Update(task)
					if err != nil {
						utils.Error("更新任务状态失败", map[string]string{
							"taskId": task.ID,
							"error":  err.Error(),
						})
						fmt.Printf("更新任务状态失败: %v\n", err)
						taskMutex.Lock()
						delete(taskBeingProcessed, task.ID)
						taskMutex.Unlock()
						return
					}
					
					// 使用Goroutine池处理任务
					err = w.goroutinePool.SubmitFunc(func() error {
						w.processTask(task)
						return nil
					})
					if err != nil {
						utils.Error("提交任务到Goroutine池失败", map[string]string{
							"taskId": task.ID,
							"error":  err.Error(),
						})
						task.Status = "failed"
						task.Error = fmt.Sprintf("提交任务失败: %v", err)
						task.Finished = time.Now()
						w.taskQueue.Update(task)
						
						taskMutex.Lock()
						delete(taskBeingProcessed, task.ID)
						taskMutex.Unlock()
					}
					return
				}
				taskMutex.Unlock()
			}
		}
	}
}

// processTask 处理单个任务
func (w *Worker) processTask(task *queue.Task) {
	utils.Info("开始处理任务", map[string]string{"taskId": task.ID})
	
	defer func() {
		// 清理任务处理标记
		taskMutex.Lock()
		delete(taskBeingProcessed, task.ID)
		taskMutex.Unlock()
		
		utils.Info("任务处理完成", map[string]string{
			"taskId": task.ID,
			"status": task.Status,
		})
	}()
	
	// 尝试将任务的Spec转换为EditSpec
	// 这里暂时简化处理，实际项目中应该使用更完善的解析方法
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		// 如果转换失败，返回错误
		task.Status = "failed"
		task.Error = "无效的视频编辑规范"
		task.Finished = time.Now()
		w.taskQueue.Update(task)
		
		utils.Error("任务处理失败：无效的视频编辑规范", map[string]string{"taskId": task.ID})
		return
	}

	// 检查任务类型
	taskType := "videoEdit" // 默认任务类型
	if taskTypeVal, exists := spec["taskType"]; exists {
		if str, ok := taskTypeVal.(string); ok {
			taskType = str
		}
	}

	// 根据任务类型处理任务
	switch taskType {
	case "materialPreprocess":
		// 处理素材预处理任务
		w.processMaterialPreprocessTask(task, spec)
	default:
		// 处理视频编辑任务（原有逻辑）
		w.processVideoEditTask(task, spec)
	}
}

// processMaterialPreprocessTask 处理素材预处理任务
func (w *Worker) processMaterialPreprocessTask(task *queue.Task, spec map[string]interface{}) {
	utils.Info("开始处理素材预处理任务", map[string]string{"taskId": task.ID})
	
	// 创建素材预处理器服务
	materialPreprocessor := NewMaterialPreprocessorService()
	
	// 处理任务
	err := materialPreprocessor.Process(task)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		task.Finished = time.Now()
		w.taskQueue.Update(task)
		
		utils.Error("素材预处理任务失败", map[string]string{
			"taskId": task.ID,
			"error":  err.Error(),
		})
		return
	}
	
	// 任务成功完成，状态已经在Process方法中更新
	task.Finished = time.Now()
	w.taskQueue.Update(task)
	
	utils.Info("素材预处理任务完成", map[string]string{"taskId": task.ID})
}

// processVideoEditTask 处理视频编辑任务
func (w *Worker) processVideoEditTask(task *queue.Task, spec map[string]interface{}) {
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
	
	// 获取编码预设配置
	encodingPreset := "medium"
	if presetVal, exists := spec["preset"]; exists {
		if str, ok := presetVal.(string); ok {
			encodingPreset = str
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
	preset := encodingPreset // 使用配置的预设
	
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
		// 重试机制
		var err error
		maxRetries := 3
		for i := 0; i <= maxRetries; i++ {
			utils.Info("开始视频合并", map[string]string{
				"taskId":   task.ID,
				"attempt":  fmt.Sprintf("%d", i+1),
				"maxRetry": fmt.Sprintf("%d", maxRetries),
			})
			
			err = w.mergeVideos(inputFiles, outPath, width, height, fps, preset)
			if err == nil {
				task.Status = "completed"
				task.Result = outPath
				
				utils.Info("视频合并成功", map[string]string{
					"taskId":  task.ID,
					"attempt": fmt.Sprintf("%d", i+1),
				})
				break
			}
			
			// 如果是最后一次重试，记录错误
			if i == maxRetries {
				task.Status = "failed"
				task.Error = fmt.Sprintf("视频合并失败（已重试%d次）: %v", maxRetries, err)
				
				utils.Error("视频合并最终失败", map[string]string{
					"taskId":   task.ID,
					"attempts": fmt.Sprintf("%d", maxRetries+1),
					"error":    err.Error(),
				})
				break
			}
			
			// 等待一段时间后重试
			utils.Warn("视频合并失败，准备重试", map[string]string{
				"taskId":   task.ID,
				"attempt":  fmt.Sprintf("%d", i+1),
				"delay":    fmt.Sprintf("%d", (i+1)*2),
				"error":    err.Error(),
			})
			fmt.Printf("视频合并失败，%d秒后进行第%d次重试: %v\n", (i+1)*2, i+1, err)
			time.Sleep(time.Duration(i+1) * 2 * time.Second)
		}
	} else {
		task.Status = "failed"
		task.Error = "缺少必要的视频编辑参数"
		
		utils.Error("任务处理失败：缺少必要的视频编辑参数", map[string]string{
			"taskId":  task.ID,
			"outPath": outPath,
			"width":   fmt.Sprintf("%d", width),
			"height":  fmt.Sprintf("%d", height),
		})
	}

	task.Finished = time.Now()
	w.taskQueue.Update(task)
}

// mergeVideos 合并视频文件
func (w *Worker) mergeVideos(inputFiles []string, outPath string, width, height, fps int, preset string) error {
	utils.Info("开始合并视频", map[string]string{
		"inputFiles": fmt.Sprintf("%v", inputFiles),
		"outPath":    outPath,
		"width":      fmt.Sprintf("%d", width),
		"height":     fmt.Sprintf("%d", height),
		"fps":        fmt.Sprintf("%d", fps),
		"preset":     preset,
	})
	
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		utils.Error("获取当前工作目录失败", map[string]string{"error": err.Error()})
		return errors.Wrap(err, "无法获取当前工作目录")
	}

	// 创建缓存键
	cacheKey := &TaskCacheKey{
		InputFiles: inputFiles,
		Width:      width,
		Height:     height,
		FPS:        fps,
		Preset:     preset,
	}
	key := cacheKey.GenerateKey()

	// 检查缓存中是否存在结果
	if entry, exists := processingCache.Get(key); exists {
		// 缓存命中，直接复制文件
		utils.Info("缓存命中", map[string]string{"cachedFile": entry.OutputFile})
		fmt.Printf("缓存命中，使用缓存结果: %s\n", entry.OutputFile)
		
		// 使用缓冲池复制文件
		if err := copyFileWithBufferPool(entry.OutputFile, outPath); err != nil {
			utils.Error("复制缓存文件失败", map[string]string{
				"source": entry.OutputFile,
				"target": outPath,
				"error":  err.Error(),
			})
			return errors.Wrap(err, "复制缓存文件失败")
		}
		
		utils.Info("缓存文件复制成功", map[string]string{
			"source": entry.OutputFile,
			"target": outPath,
		})
		return nil
	}

	// 预处理输入文件（分析视频信息，但不改变文件）
	utils.Debug("开始预处理输入文件", nil)
	_, err = videoInfoCache.PreprocessInputFiles(inputFiles, wd)
	if err != nil {
		utils.Error("预处理输入文件失败", map[string]string{"error": err.Error()})
		return errors.Wrap(err, "预处理输入文件失败")
	}

	// 并行解码输入文件
	utils.Debug("开始并行解码", nil)
	decodedFiles, err := w.parallelDecode(inputFiles, wd)
	if err != nil {
		utils.Error("并行解码失败", map[string]string{"error": err.Error()})
		return errors.Wrap(err, "并行解码失败")
	}
	
	// 记得清理临时文件
	defer func() {
		// 获取临时目录路径并删除
		if len(decodedFiles) > 0 {
			tempDir := filepath.Dir(decodedFiles[0])
			utils.Debug("清理临时文件", map[string]string{"tempDir": tempDir})
			os.RemoveAll(tempDir)
		}
	}()

	// 创建输入文件列表（使用绝对路径）
	var fullInputFiles []string
	for _, file := range decodedFiles {
		fullPath := file
		fullInputFiles = append(fullInputFiles, fullPath)
	}

	// 检查所有输入文件是否存在
	for _, file := range fullInputFiles {
		if _, err := os.Stat(file); os.IsNotExist(err) {
			utils.Error("输入文件不存在", map[string]string{"file": file})
			return errors.Wrapf(err, "输入文件不存在: %s", file)
		}
	}

	// 创建临时文件列表用于concat
	listFile := filepath.Join(wd, "video", "file_list.txt")
	utils.Debug("创建文件列表", map[string]string{"listFile": listFile})
	file, err := os.Create(listFile)
	if err != nil {
		utils.Error("创建文件列表失败", map[string]string{"error": err.Error()})
		return errors.Wrap(err, "无法创建文件列表")
	}
	defer os.Remove(listFile)

	for _, input := range fullInputFiles {
		// 在列表文件中使用双反斜杠转义路径
		fmt.Fprintf(file, "file '%s'\n", strings.ReplaceAll(input, "\\", "/"))
	}
	file.Close()

	// 选择最佳编码器
	utils.Debug("选择最佳编码器", nil)
	videoEncoder := selectBestEncoder()
	
	// 尝试使用选定的编码器，如果失败则降级
	utils.Info("开始视频编码", map[string]string{"encoder": videoEncoder})
	err = tryEncoderWithFallback(videoEncoder, listFile, outPath, width, height, fps, preset)
	if err != nil {
		utils.Error("视频编码失败", map[string]string{"error": err.Error()})
		return errors.Wrap(err, "视频编码失败")
	}

	// 检查输出文件是否存在
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		utils.Error("输出文件未生成", map[string]string{"outPath": outPath})
		return errors.New("输出文件未生成")
	}

	// 将结果添加到缓存
	fileInfo, err := os.Stat(outPath)
	if err == nil {
		entry := &CacheEntry{
			OutputFile: outPath,
			CreatedAt:  time.Now(),
			Size:       fileInfo.Size(),
		}
		processingCache.Put(key, entry)
		
		utils.Info("处理结果已缓存", map[string]string{
			"outPath": outPath,
			"size":    fmt.Sprintf("%d", fileInfo.Size()),
		})
	}

	utils.Info("视频合并完成", map[string]string{"outPath": outPath})
	return nil
}

// copyFileWithBufferPool 使用缓冲池复制文件
func copyFileWithBufferPool(src, dst string) error {
	utils.Debug("使用缓冲池复制文件", map[string]string{
		"source": src,
		"target": dst,
	})
	
	sourceFile, err := os.Open(src)
	if err != nil {
		utils.Error("打开源文件失败", map[string]string{
			"source": src,
			"error":  err.Error(),
		})
		return errors.Wrap(err, "无法打开源文件")
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		utils.Error("创建目标文件失败", map[string]string{
			"target": dst,
			"error":  err.Error(),
		})
		return errors.Wrap(err, "无法创建目标文件")
	}
	defer destFile.Close()

	// 使用缓冲池获取缓冲区
	buf := bufferPool.Get(64 * 1024) // 64KB缓冲区
	defer bufferPool.Put(buf)

	// 复制文件
	_, err = copyWithBuffer(sourceFile, destFile, buf)
	if err != nil {
		utils.Error("复制文件失败", map[string]string{
			"source": src,
			"target": dst,
			"error":  err.Error(),
		})
	}
	return err
}

// copyWithBuffer 使用指定缓冲区复制数据
func copyWithBuffer(src, dst *os.File, buf []byte) (int64, error) {
	var written int64
	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			nw, err := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if err != nil {
				utils.Error("写入文件失败", map[string]string{"error": err.Error()})
				return written, errors.Wrap(err, "写入文件失败")
			}
			if nr != nw {
				utils.Error("写入不完整", nil)
				return written, errors.New("写入不完整")
			}
		}
		if err != nil {
			if isEOF(err) {
				utils.Debug("文件复制完成", nil)
				break
			}
			utils.Error("读取文件失败", map[string]string{"error": err.Error()})
			return written, errors.Wrap(err, "读取文件失败")
		}
	}
	return written, nil
}

// ParallelDecodeForTest 用于测试的并行解码方法
func (w *Worker) ParallelDecodeForTest(inputFiles []string, workDir string) ([]string, error) {
	return w.parallelDecode(inputFiles, workDir)
}

// parallelDecode 并行解码输入文件
func (w *Worker) parallelDecode(inputFiles []string, workDir string) ([]string, error) {
	utils.Info("开始并行解码", map[string]string{
		"inputFiles": fmt.Sprintf("%v", inputFiles),
	})
	
	// 创建临时目录用于存储解码后的文件
	tempDir := filepath.Join(workDir, "temp", fmt.Sprintf("decode_%d", time.Now().UnixNano()))
	utils.Debug("创建临时目录", map[string]string{"tempDir": tempDir})
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		utils.Error("创建临时目录失败", map[string]string{"error": err.Error()})
		return nil, errors.Wrap(err, "创建临时目录失败")
	}
	
	// 使用WaitGroup等待所有解码任务完成
	var wg sync.WaitGroup
	decodedFiles := make([]string, len(inputFiles))
	errorsChan := make(chan error, len(inputFiles))
	
	// 并行解码所有输入文件
	for i, file := range inputFiles {
		wg.Add(1)
		go func(index int, inputFile string) {
			defer wg.Done()
			
			utils.Debug("开始解码单个文件", map[string]string{
				"index":     fmt.Sprintf("%d", index),
				"inputFile": inputFile,
			})
			
			// 构造完整输入文件路径
			fullInputPath := filepath.Join(workDir, "video", inputFile)
			
			// 构造输出文件路径
			outputFile := fmt.Sprintf("decoded_%d.mp4", index)
			fullOutputPath := filepath.Join(tempDir, outputFile)
			decodedFiles[index] = fullOutputPath
			
			// 使用ffmpeg解码文件
			// 这里我们直接转码为统一格式，以便后续处理
			cmd := exec.Command("ffmpeg", "-i", fullInputPath,
				"-c:v", "libx264", "-preset", "ultrafast", "-crf", "28",
				"-c:a", "aac", "-b:a", "96k",
				"-threads", "0",
				fullOutputPath, "-y")
			
			utils.Debug("执行FFmpeg解码命令", map[string]string{
				"command": fmt.Sprintf("%v", cmd.Args),
			})
			
			output, err := cmd.CombinedOutput()
			if err != nil {
				utils.Error("解码文件失败", map[string]string{
					"inputFile": inputFile,
					"error":     err.Error(),
					"output":    string(output),
				})
				errorsChan <- errors.Wrapf(err, "解码文件 %s 失败, 输出: %s", inputFile, string(output))
				return
			}
			
			utils.Debug("文件解码成功", map[string]string{
				"inputFile": inputFile,
				"outputFile": fullOutputPath,
			})
		}(i, file)
	}
	
	// 等待所有解码任务完成
	wg.Wait()
	close(errorsChan)
	
	// 检查是否有错误
	if len(errorsChan) > 0 {
		// 清理临时目录
		utils.Warn("解码过程中出现错误，清理临时目录", map[string]string{"tempDir": tempDir})
		os.RemoveAll(tempDir)
		err := <-errorsChan
		utils.Error("并行解码失败", map[string]string{"error": err.Error()})
		return nil, err
	}
	
	utils.Info("并行解码完成", map[string]string{"decodedFiles": fmt.Sprintf("%v", decodedFiles)})
	return decodedFiles, nil
}

// 检查错误是否是EOF
func isEOF(err error) bool {
	return err != nil && (err == io.EOF || strings.Contains(err.Error(), "EOF"))
}

// 使用指定编码器运行FFmpeg
func runFFmpegWithEncoder(encoder, listFile, outPath string, width, height, fps int, preset string) error {
	utils.Debug("使用指定编码器运行FFmpeg", map[string]string{
		"encoder": encoder,
		"width":   fmt.Sprintf("%d", width),
		"height":  fmt.Sprintf("%d", height),
		"fps":     fmt.Sprintf("%d", fps),
		"preset":  preset,
	})
	
	// 构建命令
	var cmd *exec.Cmd
	
	// 根据编码器类型选择合适的预设
	encoderPreset := preset
	if encoder != "libx264" {
		// 硬件编码器通常支持更少的预设选项
		encoderPreset = "fast" // 大多数硬件编码器都支持fast预设
	}
	
	// 对于不同编码器，使用不同的优化参数
	switch encoder {
	case "h264_nvenc":
		// NVENC编码器不支持CRF模式，使用cq模式
		cmd = exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFile,
			"-vf", fmt.Sprintf("scale=%d:%d,fps=%d", width, height, fps),
			"-c:v", encoder, "-cq", "28", "-preset", encoderPreset,
			"-c:a", "aac", "-b:a", "96k", // 降低音频比特率
			"-threads", "0", // 自动选择线程数
			outPath, "-y")
	case "libx264":
		// libx264编码器使用CRF模式
		cmd = exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFile,
			"-vf", fmt.Sprintf("scale=%d:%d,fps=%d", width, height, fps),
			"-c:v", encoder, "-crf", "28", "-preset", encoderPreset,
			"-c:a", "aac", "-b:a", "96k", // 降低音频比特率
			"-threads", "0", // 自动选择线程数
			outPath, "-y")
	default:
		// 其他编码器使用通用参数
		cmd = exec.Command("ffmpeg", "-f", "concat", "-safe", "0", "-i", listFile,
			"-vf", fmt.Sprintf("scale=%d:%d,fps=%d", width, height, fps),
			"-c:v", encoder, "-crf", "28", "-preset", encoderPreset,
			"-c:a", "aac", "-b:a", "96k", // 降低音频比特率
			"-threads", "0", // 自动选择线程数
			outPath, "-y")
	}
	
	utils.Debug("执行FFmpeg命令", map[string]string{"command": fmt.Sprintf("%v", cmd.Args)})
	
	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		utils.Error("FFmpeg执行失败", map[string]string{
			"encoder": encoder,
			"error":   err.Error(),
			"output":  string(output),
		})
		return errors.Wrapf(err, "ffmpeg执行失败, 输出: %s", string(output))
	}
	
	utils.Debug("FFmpeg执行成功", map[string]string{"encoder": encoder})
	return nil
}
