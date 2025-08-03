package service

import (
	"time"

	"github.com/google/uuid"
	"github.com/u2takey/ffmpeg-go/api"
)

// VideoEditor 视频编辑服务接口
type VideoEditor interface {
	SubmitTask(request *api.VideoEditRequest) (*Task, error)
	GetTaskStatus(taskID string) (*Task, error)
	CancelTask(taskID string) error
}

// VideoEditorService 视频编辑服务实现
type VideoEditorService struct {
	taskQueue TaskQueue
}

// NewVideoEditorService 创建视频编辑服务实例
func NewVideoEditorService(taskQueue TaskQueue) *VideoEditorService {
	return &VideoEditorService{
		taskQueue: taskQueue,
	}
}

// SubmitTask 提交视频编辑任务
func (s *VideoEditorService) SubmitTask(request *api.VideoEditRequest) (*Task, error) {
	// 创建任务
	task := &Task{
		ID:       uuid.New().String(),
		Spec:     request.Spec,
		Status:   "pending",
		Created:  time.Now(),
		Progress: 0.0,
	}

	// 添加到任务队列
	err := s.taskQueue.Add(task)
	if err != nil {
		return nil, err
	}

	return task, nil
}

// GetTaskStatus 获取任务状态
func (s *VideoEditorService) GetTaskStatus(taskID string) (*Task, error) {
	// TODO: 实现获取任务状态逻辑
	return s.taskQueue.Get(taskID)
}

// CancelTask 取消任务
func (s *VideoEditorService) CancelTask(taskID string) error {
	// 获取任务
	task, err := s.taskQueue.Get(taskID)
	if err != nil {
		return err
	}

	if task == nil {
		return nil // 任务不存在，视为取消成功
	}

	// 如果任务已经在处理中或已完成，则不能取消
	if task.Status == "processing" || task.Status == "completed" || task.Status == "failed" {
		return nil
	}

	// 更新任务状态为已取消
	task.Status = "cancelled"
	task.Finished = time.Now()
	return s.taskQueue.Update(task)
}

// ListTasks 列出所有任务
func (s *VideoEditorService) ListTasks() ([]*Task, error) {
	// TODO: 实现列出所有任务逻辑
	return s.taskQueue.List()
}