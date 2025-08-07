package service

import (
	"time"
	
	"github.com/u2takey/ffmpeg-go/queue"
)

// VideoEditRequest 视频编辑请求
type VideoEditRequest struct {
	Spec     interface{}  `json:"spec"`
	Priority queue.TaskPriority `json:"priority,omitempty"` // 添加优先级字段
}

// VideoEditor 视频编辑服务接口
type VideoEditor interface {
	SubmitTask(req *VideoEditRequest) (*queue.Task, error)
	GetTaskStatus(taskID string) (*queue.Task, error)
	CancelTask(taskID string) error
	// 移除了GetTaskLogs方法，因为TaskLogger没有GetLogs方法
}

// VideoEditorService 视频编辑服务实现
type VideoEditorService struct {
	taskQueue queue.TaskQueue
}

// NewVideoEditorService 创建新的视频编辑服务
func NewVideoEditorService(taskQueue queue.TaskQueue) VideoEditor {
	return &VideoEditorService{
		taskQueue: taskQueue,
	}
}

// SubmitTask 提交视频编辑任务
func (s *VideoEditorService) SubmitTask(req *VideoEditRequest) (*queue.Task, error) {
	// 创建任务请求
	taskReq := &queue.Task{
		Spec:      req.Spec,
		Status:    "pending",
		Created:   time.Now(),
		Priority:  req.Priority, // 设置任务优先级
	}
	
	// 添加任务到队列
	err := s.taskQueue.Push(taskReq)
	if err != nil {
		return nil, err
	}
	
	// 创建任务日志记录器
	taskLogger, err := NewTaskLogger(taskReq.ID)
	if err != nil {
		// 即使日志记录器创建失败，也不影响任务提交
		return taskReq, nil
	}
	
	// 记录任务提交日志
	taskLogger.Log("INFO", "任务已提交到队列", map[string]interface{}{
		"taskId":   taskReq.ID,
		"priority": taskReq.Priority,
		"status":   taskReq.Status,
	})
	
	return taskReq, nil
}

// GetTaskStatus 获取任务状态
func (s *VideoEditorService) GetTaskStatus(taskID string) (*queue.Task, error) {
	return s.taskQueue.Get(taskID)
}

// CancelTask 取消任务
func (s *VideoEditorService) CancelTask(taskID string) error {
	task, err := s.taskQueue.Get(taskID)
	if err != nil {
		return err
	}
	
	if task == nil {
		return nil
	}
	
	// 只能取消待处理和处理中的任务
	if task.Status == "pending" || task.Status == "processing" {
		task.Status = "cancelled"
		task.Finished = time.Now()
		return s.taskQueue.Update(task)
	}
	
	return nil
}