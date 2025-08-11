package service

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
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
		if task.Verbose {
			log.Printf("Worker %d: Processing task %s", w.id, task.ID)
		}
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

		if task.Verbose {
			log.Printf("Worker %d: Completed task %s", w.id, task.ID)
		}
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
		// 如果没有指定任务类型，则尝试作为视频编辑任务处理
		return w.processVideoEditTask(task)
	}

	taskType, ok := spec["taskType"].(string)
	if !ok {
		// 如果没有指定任务类型，则尝试作为视频编辑任务处理
		return w.processVideoEditTask(task)
	}

	switch taskType {
	case "materialPreprocess":
		// 处理素材预处理任务
		preprocessor := service.NewMaterialPreprocessorService()
		return preprocessor.Process(task)
	default:
		// 默认作为视频编辑任务处理
		return w.processVideoEditTask(task)
	}
}

// processVideoEditTask 处理视频编辑任务
func (w *WorkerImpl) processVideoEditTask(task *queue.Task) error {
	task.Status = "processing"
	
	// 将任务规范转换为EditSpec
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid task spec format")
	}
	
	// 创建EditSpec对象
	editSpec := &EditSpec{}
	
	// 从map转换到EditSpec结构体
	if outPath, ok := spec["outPath"].(string); ok {
		editSpec.OutPath = outPath
	} else {
		// 如果没有指定输出路径，使用默认路径
		editSpec.OutPath = fmt.Sprintf("./output/%s.mp4", task.ID)
	}
	
	if width, ok := spec["width"].(float64); ok {
		editSpec.Width = int(width)
	} else {
		editSpec.Width = 1920 // 默认宽度
	}
	
	if height, ok := spec["height"].(float64); ok {
		editSpec.Height = int(height)
	} else {
		editSpec.Height = 1080 // 默认高度
	}
	
	if fps, ok := spec["fps"].(float64); ok {
		editSpec.Fps = int(fps)
	} else {
		editSpec.Fps = 30 // 默认帧率
	}
	
	if verbose, ok := spec["verbose"].(bool); ok {
		editSpec.Verbose = verbose
	} else {
		editSpec.Verbose = task.Verbose // 使用任务级别的verbose设置
	}
	
	// 处理clips
	if clips, ok := spec["clips"].([]interface{}); ok {
		editSpec.Clips = make([]*Clip, len(clips))
		for i, clip := range clips {
			if clipMap, ok := clip.(map[string]interface{}); ok {
				editSpec.Clips[i] = &Clip{}
				
				// 处理layers
				if layers, ok := clipMap["layers"].([]interface{}); ok {
					editSpec.Clips[i].Layers = make([]*Layer, len(layers))
					for j, layer := range layers {
						if layerMap, ok := layer.(map[string]interface{}); ok {
							editSpec.Clips[i].Layers[j] = &Layer{}
							
							if layerType, ok := layerMap["type"].(string); ok {
								editSpec.Clips[i].Layers[j].Type = layerType
							}
							
							if path, ok := layerMap["path"].(string); ok {
								editSpec.Clips[i].Layers[j].Path = path
							}
							
							if text, ok := layerMap["text"].(string); ok {
								editSpec.Clips[i].Layers[j].Text = text
							}
						}
					}
				}
			}
		}
	} else {
		return fmt.Errorf("clips字段缺失或格式不正确")
	}
	
	// 创建任务日志记录器
	taskLogger, _ := service.NewTaskLogger(task.ID)
	if taskLogger != nil && editSpec.Verbose {
		taskLogger.Log("INFO", "开始处理视频编辑任务", map[string]interface{}{
			"taskId": task.ID,
			"clips":  len(editSpec.Clips),
		})
	}
	
	// 创建Editly实例并执行编辑
	editly := NewEditly(editSpec)
	
	err := editly.Edit()
	if err != nil {
		if taskLogger != nil && editSpec.Verbose {
			taskLogger.Log("ERROR", "视频编辑任务失败", map[string]interface{}{
				"taskId": task.ID,
				"error":  err.Error(),
			})
		}
		return err
	}
	
	if taskLogger != nil && editSpec.Verbose {
		taskLogger.Log("INFO", "视频编辑任务完成", map[string]interface{}{
			"taskId":  task.ID,
			"outPath": editSpec.OutPath,
		})
	}
	
	task.Progress = 1.0
	return nil
}

// EditSpec 视频编辑规范
type EditSpec struct {
	OutPath        string                 `json:"outPath"`
	Width          int                    `json:"width"`
	Height         int                    `json:"height"`
	Fps            int                    `json:"fps"`
	Defaults       map[string]interface{} `json:"defaults,omitempty"`
	Clips          []*Clip                `json:"clips"`
	AudioTracks    []string               `json:"audioTracks,omitempty"`
	KeepSourceAudio bool                 `json:"keepSourceAudio,omitempty"`
	Verbose        bool                   `json:"verbose,omitempty"` // 添加详细日志开关
}

// Clip 视频片段
type Clip struct {
	Layers []*Layer `json:"layers"`
}

// Layer 视频层
type Layer struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	Text string `json:"text,omitempty"`
}

// NewEditly 创建新的视频编辑器
func NewEditly(spec *EditSpec) *Editly {
	return &Editly{
		spec: spec,
	}
}

// Editly 视频编辑器
type Editly struct {
	spec *EditSpec
}

// Edit 执行视频编辑
func (e *Editly) Edit() error {
	if e.spec.Verbose {
		log.Printf("开始编辑视频: %s", e.spec.OutPath)
		log.Printf("尺寸: %dx%d, 帧率: %d", e.spec.Width, e.spec.Height, e.spec.Fps)
	}

	// 创建输出目录
	dir := filepath.Dir(e.spec.OutPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 验证输入文件
	if err := e.validateInputs(); err != nil {
		return fmt.Errorf("输入验证失败: %w", err)
	}

	// 处理每个片段
	//tempFiles, err := e.processClips()
	//if err != nil {
	//	return fmt.Errorf("处理片段失败: %w", err)
	//}
	//defer e.cleanupTempFiles(tempFiles)

	// 连接片段
	//if err := e.concatenateClips(tempFiles, e.spec.OutPath); err != nil {
	//	return fmt.Errorf("连接片段失败: %w", err)
	//}

	if e.spec.Verbose {
		log.Printf("视频编辑完成: %s", e.spec.OutPath)
	}

	return nil
}

// validateInputs 验证输入文件
func (e *Editly) validateInputs() error {
	if e.spec.Verbose {
		log.Println("验证输入文件...")
	}

	for i, clip := range e.spec.Clips {
		for j, layer := range clip.Layers {
			if layer.Type == "video" && layer.Path != "" {
				if e.spec.Verbose {
					log.Printf("验证片段 %d, 层 %d: %s", i+1, j+1, layer.Path)
				}
			}
		}
	}

	return nil
}

