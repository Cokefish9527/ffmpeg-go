package api

import (
	"time"
	
	"github.com/u2takey/ffmpeg-go/service"
)

// TaskPriority 任务优先级
type TaskPriority int

const (
	PriorityLow TaskPriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// VideoEditRequest 视频编辑请求
type VideoEditRequest struct {
	Spec     interface{}  `json:"spec"`
	Priority TaskPriority `json:"priority,omitempty"` // 添加优先级字段
}

// TaskStatusResponse 任务状态响应
type TaskStatusResponse struct {
	TaskID    string       `json:"taskId"`
	Status    string       `json:"status"`
	Progress  float64      `json:"progress"`
	Message   string       `json:"message,omitempty"`
	Created   string       `json:"created,omitempty"`
	Started   string       `json:"started,omitempty"`
	Finished  string       `json:"finished,omitempty"`
	OutputURL string       `json:"outputUrl,omitempty"`
	Priority  TaskPriority `json:"priority,omitempty"` // 添加优先级字段
}

// ConvertTaskToResponse 将任务转换为响应格式
func ConvertTaskToResponse(task *service.Task) *TaskStatusResponse {
	response := &TaskStatusResponse{
		TaskID:   task.ID,
		Status:   task.Status,
		Progress: task.Progress,
		Message:  task.Error,
		Priority: task.Priority, // 添加优先级字段
	}
	
	if !task.Created.IsZero() {
		response.Created = task.Created.Format(time.RFC3339)
	}
	
	if !task.Started.IsZero() {
		response.Started = task.Started.Format(time.RFC3339)
	}
	
	if !task.Finished.IsZero() {
		response.Finished = task.Finished.Format(time.RFC3339)
	}
	
	if task.Result != "" {
		response.OutputURL = task.Result
	}
	
	return response
}