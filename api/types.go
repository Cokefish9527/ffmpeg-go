package api

// VideoEditRequest 视频编辑请求结构体
type VideoEditRequest struct {
	Spec       interface{} `json:"spec"`
	OutputPath string      `json:"outputPath,omitempty"` // 本地输出路径
	OSSOutput  *OSSOutput  `json:"ossOutput,omitempty"`  // OSS 输出配置
}

// OSSOutput 阿里云OSS输出配置
type OSSOutput struct {
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

// VideoEditResponse 视频编辑响应结构体
type VideoEditResponse struct {
	TaskID    string `json:"taskId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	OutputURL string `json:"outputUrl,omitempty"`
}

// TaskStatusResponse 任务状态响应结构体
type TaskStatusResponse struct {
	TaskID    string  `json:"taskId"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	Message   string  `json:"message,omitempty"`
	Created   string  `json:"created,omitempty"`
	Started   string  `json:"started,omitempty"`
	Finished  string  `json:"finished,omitempty"`
	OutputURL string  `json:"outputUrl,omitempty"`
}