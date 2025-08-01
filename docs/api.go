package docs

import (
	"github.com/u2takey/ffmpeg-go/api"
)

// SubmitVideoEditRequest 提交视频编辑任务请求
//
// swagger:parameters submitVideoEdit
type SubmitVideoEditRequest struct {
	// in: body
	Body api.VideoEditRequest
}

// SubmitVideoEditResponse 提交视频编辑任务响应
//
// swagger:response submitVideoEditResponse
type SubmitVideoEditResponse struct {
	// in: body
	Body api.VideoEditResponse
}

// BadRequestResponse 请求格式错误响应
//
// swagger:response badRequestResponse
type BadRequestResponse struct {
	// in: body
	Body struct {
		Error string `json:"error"`
	}
}

// InternalServerErrorResponse 服务器内部错误响应
//
// swagger:response internalServerErrorResponse
type InternalServerErrorResponse struct {
	// in: body
	Body struct {
		Error string `json:"error"`
	}
}

// TaskStatusResponse 任务状态响应
//
// swagger:response taskStatusResponse
type TaskStatusResponse struct {
	// in: body
	Body api.TaskStatusResponse
}

// TaskNotFoundResponse 任务未找到响应
//
// swagger:response taskNotFoundResponse
type TaskNotFoundResponse struct {
	// in: body
	Body struct {
		Error string `json:"error"`
	}
}

// CancelTaskResponse 取消任务响应
//
// swagger:response cancelTaskResponse
type CancelTaskResponse struct {
	// in: body
	Body struct {
		Message string `json:"message"`
		TaskId  string `json:"taskId"`
	}
}