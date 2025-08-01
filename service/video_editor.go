package service

import (
	"github.com/google/uuid"
)

// VideoEditor 视频编辑服务接口
type VideoEditor interface {
	// SubmitTask 提交视频编辑任务
	SubmitTask(spec interface{}) (string, error)
	
	// GetTaskStatus 获取任务状态
	GetTaskStatus(taskID string) (*Task, error)
	
	// CancelTask 取消任务
	CancelTask(taskID string) error
	
	// ListTasks 列出所有任务
	ListTasks() ([]*Task, error)
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
func (s *VideoEditorService) SubmitTask(spec interface{}) (string, error) {
	// 生成任务ID
	taskID := uuid.New().String()
	
	// 创建任务对象
	task := &Task{
		ID:       taskID,
		Spec:     spec,
		Status:   "pending",
		Progress: 0.0,
	}
	
	// 将任务添加到任务队列
	err := s.taskQueue.Add(task)
	if err != nil {
		return "", err
	}
	
	return taskID, nil
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
		return nil // 任务不存在，直接返回
	}
	
	// 只有在任务未完成时才能取消
	if task.Status == "pending" || task.Status == "processing" {
		task.Status = "cancelled"
		task.Error = "任务已被取消"
		return s.taskQueue.Update(task)
	}
	
	return nil
}

// ListTasks 列出所有任务
func (s *VideoEditorService) ListTasks() ([]*Task, error) {
	// TODO: 实现列出所有任务逻辑
	return s.taskQueue.List()
}