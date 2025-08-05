package api

import (
	"github.com/u2takey/ffmpeg-go/service"
)

// VideoEditRequest 视频编辑请求结构体
type VideoEditRequest struct {
	// Spec 视频编辑规范，包含具体的编辑参数和配置
	Spec       interface{}        `json:"spec"`
	// OutputPath 本地输出路径，指定视频文件保存的本地路径
	OutputPath string             `json:"outputPath,omitempty"` // 本地输出路径
	// OSSOutput 阿里云OSS输出配置，用于将输出文件上传到OSS
	OSSOutput  *OSSOutput         `json:"ossOutput,omitempty"`  // OSS 输出配置
	// Priority 任务优先级
	Priority   service.TaskPriority `json:"priority,omitempty"`   // 任务优先级
}

// OSSOutput 阿里云OSS输出配置
type OSSOutput struct {
	// Bucket OSS存储桶名称
	Bucket    string `json:"bucket"`
	// Key OSS对象键名，即文件在OSS中的路径
	Key       string `json:"key"`
	// Endpoint OSS服务终端节点
	Endpoint  string `json:"endpoint"`
	// AccessKey 阿里云访问密钥ID
	AccessKey string `json:"accessKey"`
	// SecretKey 阿里云访问密钥Secret
	SecretKey string `json:"secretKey"`
}

// VideoEditResponse 视频编辑响应结构体
type VideoEditResponse struct {
	// TaskID 任务ID，用于标识和查询任务状态
	TaskID    string `json:"taskId"`
	// Status 任务状态，如"accepted"表示已接受
	Status    string `json:"status"`
	// Message 响应消息，提供额外的信息说明
	Message   string `json:"message"`
	// OutputURL 输出文件的URL地址（如果已上传到OSS）
	OutputURL string `json:"outputUrl,omitempty"`
}

// TaskStatusResponse 任务状态响应结构体
type TaskStatusResponse struct {
	// TaskID 任务ID
	TaskID    string             `json:"taskId"`
	// Status 任务当前状态，如"pending"、"processing"、"completed"、"failed"
	Status    string             `json:"status"`
	// Progress 任务进度百分比，范围0-1
	Progress  float64            `json:"progress"`
	// Message 任务相关的消息或错误信息
	Message   string             `json:"message,omitempty"`
	// Created 任务创建时间，RFC3339格式
	Created   string             `json:"created,omitempty"`
	// Started 任务开始处理时间，RFC3339格式
	Started   string             `json:"started,omitempty"`
	// Finished 任务完成时间，RFC3339格式
	Finished  string             `json:"finished,omitempty"`
	// OutputURL 输出文件的URL地址
	OutputURL string             `json:"outputUrl,omitempty"`
	// Priority 任务优先级
	Priority  service.TaskPriority `json:"priority,omitempty"`   // 任务优先级
}

// MaterialUploadResponse 素材上传响应结构体
type MaterialUploadResponse struct {
	// TaskID 任务ID，用于标识和查询转换任务状态
	TaskID    string `json:"taskId"`
	// Status 任务状态
	Status    string `json:"status"`
	// Message 响应消息
	Message   string `json:"message"`
}

// VideoURLRequest 视频URL请求结构体
type VideoURLRequest struct {
	// URL 视频文件的URL地址
	URL string `json:"url"`
}

// VideoURLResponse 视频URL响应结构体
type VideoURLResponse struct {
	// Status 处理状态
	Status string `json:"status"`
	// Message 响应消息
	Message string `json:"message"`
	// TSFilePath 转换后的TS文件本地路径
	TSFilePath string `json:"tsFilePath,omitempty"`
	// Error 错误信息（如果有）
	Error string `json:"error,omitempty"`
}