package api

// VideoEditRequest 视频编辑请求结构体
type VideoEditRequest struct {
	Spec       interface{} `json:"spec"`       // 视频编辑规范，与ffmpeg.EditSpec对应
	OutputPath string      `json:"outputPath"` // 本地输出路径
	OSSOutput  *OSSOutput  `json:"ossOutput"`  // OSS 输出配置
}

// OSSOutput OSS输出配置
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
	OutputURL string `json:"outputUrl"`
}

// TaskStatusResponse 任务状态响应结构体
type TaskStatusResponse struct {
	TaskID    string  `json:"taskId"`
	Status    string  `json:"status"`
	Progress  float64 `json:"progress"`
	Message   string  `json:"message"`
	Created   string  `json:"created"`
	Started   string  `json:"started"`
	Finished  string  `json:"finished"`
	OutputURL string  `json:"outputUrl"`
}