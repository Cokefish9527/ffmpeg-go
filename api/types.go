package api

// VideoEditRequest 视频编辑请求
// @Description 视频编辑请求参数
type VideoEditRequest struct {
	// 任务规格
	Spec interface{} `json:"spec"`
	// 输出路径
	OutputPath string `json:"outputPath,omitempty"`
	// OSS输出配置
	OSSOutput *OSSOutput `json:"ossOutput,omitempty"`
	// 任务优先级
	Priority int `json:"priority,omitempty"`
}

// VideoEditResponse 视频编辑响应
// @Description 视频编辑任务提交响应
type VideoEditResponse struct {
	// 任务ID
	TaskID string `json:"taskId"`
	// 状态
	Status string `json:"status"`
	// 消息
	Message string `json:"message"`
	// 输出URL
	OutputURL string `json:"outputUrl,omitempty"`
}

// OSSOutput OSS输出配置
// @Description OSS输出配置参数
type OSSOutput struct {
	// Endpoint
	Endpoint string `json:"endpoint"`
	// AccessKey
	AccessKey string `json:"accessKey"`
	// SecretKey
	SecretKey string `json:"secretKey"`
	// Bucket
	Bucket string `json:"bucket"`
	// Key
	Key string `json:"key"`
}

// TaskStatusResponse 任务状态响应
// @Description 任务状态响应信息
type TaskStatusResponse struct {
	// 任务ID
	TaskID string `json:"taskId"`
	// 状态
	Status string `json:"status"`
	// 进度
	Progress float64 `json:"progress"`
	// 消息
	Message string `json:"message,omitempty"`
	// 创建时间
	Created string `json:"created,omitempty"`
	// 开始时间
	Started string `json:"started,omitempty"`
	// 完成时间
	Finished string `json:"finished,omitempty"`
	// 输出URL
	OutputURL string `json:"outputUrl,omitempty"`
	// 优先级
	Priority int `json:"priority,omitempty"`
}

// VideoURLRequest 视频URL请求结构体
// @Description 视频URL请求参数
type VideoURLRequest struct {
	// 视频URL
	URL string `json:"url"`
}

// VideoURLResponse 视频URL响应结构体
// @Description 视频URL处理响应
type VideoURLResponse struct {
	// 状态
	Status string `json:"status"`
	// 消息
	Message string `json:"message"`
	// TS文件路径
	TSFilePath string `json:"tsFilePath,omitempty"`
	// 错误信息
	Error string `json:"error,omitempty"`
	// 任务ID
	TaskID string `json:"taskId,omitempty"`
}