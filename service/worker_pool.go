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
	
	"github.com/u2takey/ffmpeg-go/queue"
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
}

// NewWorkerPool 创建新的工作池
func NewWorkerPool(maxWorkers int, taskQueue queue.TaskQueue) *WorkerPool {
	// 如果未指定最大工作线程数，则使用CPU核心数
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	return &WorkerPool{
		workers:    make([]*Worker, 0),
		maxWorkers: maxWorkers,
		taskQueue:  taskQueue,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start 启动工作池
func (wp *WorkerPool) Start() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	// 初始化并启动工作线程
	for i := 0; i < wp.maxWorkers; i++ {
		worker := NewWorker(wp.taskQueue)
		wp.workers = append(wp.workers, worker)
		
		wp.wg.Add(1)
		go func(w *Worker) {
			defer wp.wg.Done()
			w.Start(wp.ctx)
		}(worker)
	}
	
	fmt.Printf("WorkerPool started with %d workers\n", wp.maxWorkers)
}

// Stop 停止工作池
func (wp *WorkerPool) Stop() {
	wp.mutex.Lock()
	defer wp.mutex.Unlock()
	
	// 取消上下文，通知所有工作线程停止
	wp.cancel()
	
	// 等待所有工作线程完成
	wp.wg.Wait()
	
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
	
	if newSize > currentSize {
		// 增加Worker数量
		for i := currentSize; i < newSize; i++ {
			worker := NewWorker(wp.taskQueue)
			wp.workers = append(wp.workers, worker)
			
			wp.wg.Add(1)
			go func(w *Worker) {
				defer wp.wg.Done()
				w.Start(wp.ctx)
			}(worker)
		}
		fmt.Printf("WorkerPool resized from %d to %d workers (added %d workers)\n", currentSize, newSize, newSize-currentSize)
	} else if newSize < currentSize {
		// 减少Worker数量（这里简化处理，实际项目中应该更优雅地处理）
		wp.workers = wp.workers[:newSize]
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

// 检测可用的硬件编码器
func detectHardwareEncoders() map[string]bool {
	encoders := make(map[string]bool)
	
	// 检测NVIDIA NVENC
	cmd := exec.Command("ffmpeg", "-h", "encoder=h264_nvenc")
	if err := cmd.Run(); err == nil {
		encoders["h264_nvenc"] = true
	}
	
	// 检测Intel Quick Sync
	cmd = exec.Command("ffmpeg", "-h", "encoder=h264_qsv")
	if err := cmd.Run(); err == nil {
		encoders["h264_qsv"] = true
	}
	
	// 检测AMD VCE
	cmd = exec.Command("ffmpeg", "-h", "encoder=h264_amf")
	if err := cmd.Run(); err == nil {
		encoders["h264_amf"] = true
	}
	
	return encoders
}

// 选择最佳编码器
func selectBestEncoder() string {
	availableEncoders := detectHardwareEncoders()
	
	// 优先级顺序：NVENC > QSV > AMF > libx264
	if availableEncoders["h264_nvenc"] {
		return "h264_nvenc"
	}
	
	if availableEncoders["h264_qsv"] {
		return "h264_qsv"
	}
	
	if availableEncoders["h264_amf"] {
		return "h264_amf"
	}
	
	// 默认使用libx264
	return "libx264"
}

// 尝试使用指定编码器，如果失败则降级到libx264
func tryEncoderWithFallback(encoder, listFile, outPath string, width, height, fps int, preset string) error {
	// 首先尝试使用指定的编码器
	err := runFFmpegWithEncoder(encoder, listFile, outPath, width, height, fps, preset)
	if err == nil {
		return nil // 成功则直接返回
	}
	
	// 如果失败且不是libx264，则尝试降级到libx264
	if encoder != "libx264" {
		fmt.Printf("使用编码器 %s 失败，降级到 libx264: %v\n", encoder, err)
		return runFFmpegWithEncoder("libx264", listFile, outPath, width, height, fps, preset)
	}
	
	// 如果已经是libx264还失败，则返回错误
	return err
}

// 检查错误是否是EOF
func isEOF(err error) bool {
	return err != nil && err == io.EOF
}

// 使用指定编码器运行FFmpeg
func runFFmpegWithEncoder(encoder, listFile, outPath string, width, height, fps int, preset string) error {
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
	
	// 执行命令
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("ffmpeg执行失败: %v, 输出: %s", err, string(output))
	}
	
	return nil
}

// Worker 工作者结构
type Worker struct {
	id        int
	taskQueue queue.TaskQueue
}

// NewWorker 创建新的工作者
func NewWorker(taskQueue queue.TaskQueue) *Worker {
	return &Worker{
		id:        0, // 实际项目中应该分配唯一ID
		taskQueue: taskQueue,
	}
}

// Start 启动工作者
func (w *Worker) Start(ctx context.Context) {
	// 持续处理任务直到上下文被取消
	for {
		select {
		case <-ctx.Done():
			// 上下文被取消，退出循环
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
}

// processTask 处理单个任务
func (w *Worker) processTask(task *queue.Task) {
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
		fmt.Printf("缓存命中，使用缓存结果: %s\n", entry.OutputFile)
		
		// 使用缓冲池复制文件
		if err := copyFileWithBufferPool(entry.OutputFile, outPath); err != nil {
			return fmt.Errorf("复制缓存文件失败: %v", err)
		}
		
		return nil
	}

	// 预处理输入文件（分析视频信息，但不改变文件）
	_, err = videoInfoCache.PreprocessInputFiles(inputFiles, wd)
	if err != nil {
		return fmt.Errorf("预处理输入文件失败: %v", err)
	}

	// 并行解码输入文件
	decodedFiles, err := w.parallelDecode(inputFiles, wd)
	if err != nil {
		return fmt.Errorf("并行解码失败: %v", err)
	}
	
	// 记得清理临时文件
	defer func() {
		// 获取临时目录路径并删除
		if len(decodedFiles) > 0 {
			tempDir := filepath.Dir(decodedFiles[0])
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

	// 选择最佳编码器
	videoEncoder := selectBestEncoder()
	
	// 尝试使用选定的编码器，如果失败则降级
	err = tryEncoderWithFallback(videoEncoder, listFile, outPath, width, height, fps, preset)
	if err != nil {
		return fmt.Errorf("视频编码失败: %v", err)
	}

	// 检查输出文件是否存在
	if _, err := os.Stat(outPath); os.IsNotExist(err) {
		return fmt.Errorf("输出文件未生成")
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
	}

	return nil
}

// copyFileWithBufferPool 使用缓冲池复制文件
func copyFileWithBufferPool(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 使用缓冲池获取缓冲区
	buf := bufferPool.Get(64 * 1024) // 64KB缓冲区
	defer bufferPool.Put(buf)

	// 复制文件
	_, err = copyWithBuffer(sourceFile, destFile, buf)
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
				return written, err
			}
			if nr != nw {
				return written, fmt.Errorf("写入不完整")
			}
		}
		if err != nil {
			if isEOF(err) {
				break
			}
			return written, err
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
	// 创建临时目录用于存储解码后的文件
	tempDir := filepath.Join(workDir, "temp", fmt.Sprintf("decode_%d", time.Now().UnixNano()))
	if err := os.MkdirAll(tempDir, 0755); err != nil {
		return nil, fmt.Errorf("创建临时目录失败: %v", err)
	}
	
	// 使用WaitGroup等待所有解码任务完成
	var wg sync.WaitGroup
	decodedFiles := make([]string, len(inputFiles))
	errors := make(chan error, len(inputFiles))
	
	// 并行解码所有输入文件
	for i, file := range inputFiles {
		wg.Add(1)
		go func(index int, inputFile string) {
			defer wg.Done()
			
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
			
			output, err := cmd.CombinedOutput()
			if err != nil {
				errors <- fmt.Errorf("解码文件 %s 失败: %v, 输出: %s", inputFile, err, string(output))
				return
			}
		}(i, file)
	}
	
	// 等待所有解码任务完成
	wg.Wait()
	close(errors)
	
	// 检查是否有错误
	if len(errors) > 0 {
		// 清理临时目录
		os.RemoveAll(tempDir)
		return nil, <-errors
	}
	
	return decodedFiles, nil
}