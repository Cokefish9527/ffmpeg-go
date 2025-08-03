package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	// "github.com/google/uuid"
)

// VideoEditRequest 视频编辑请求
type VideoEditRequest struct {
	Spec       interface{} `json:"spec"`
	OutputPath string      `json:"outputPath,omitempty"`
	OSSOutput  *OSSOutput  `json:"ossOutput,omitempty"`
}

// OSSOutput OSS输出配置
type OSSOutput struct {
	Bucket    string `json:"bucket"`
	Key       string `json:"key"`
	Endpoint  string `json:"endpoint"`
	AccessKey string `json:"accessKey"`
	SecretKey string `json:"secretKey"`
}

// VideoEditResponse 视频编辑响应
type VideoEditResponse struct {
	TaskID    string `json:"taskId"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	OutputURL string `json:"outputUrl,omitempty"`
}

// TaskStatusResponse 任务状态响应
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

// APILog API调用日志
type APILog struct {
	Timestamp   time.Time   `json:"timestamp"`
	Method      string      `json:"method"`
	URL         string      `json:"url"`
	Request     interface{} `json:"request,omitempty"`
	Response    interface{} `json:"response,omitempty"`
	StatusCode  int         `json:"statusCode,omitempty"`
	Error       string      `json:"error,omitempty"`
}

// VideoInfo 视频信息
type VideoInfo struct {
	FileName   string  `json:"fileName"`
	FileSize   int64   `json:"fileSize"`
	Duration   float64 `json:"duration"`
	Codec      string  `json:"codec"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	FPS        float64 `json:"fps"`
	Bitrate    int     `json:"bitrate"`
}

// VideoMergeTest 视频合并测试结构
type VideoMergeTest struct {
	baseURL    string
	logFile    *os.File
	httpClient *http.Client
	logs       []APILog
	startTime  time.Time
	inputFiles []string  // 添加输入文件列表字段
}

// NewVideoMergeTest 创建新的视频合并测试实例
func NewVideoMergeTest(baseURL string, inputFiles []string) (*VideoMergeTest, error) {
	// 确保log目录存在
	logDir := "log"
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}

	// 创建日志文件
	logFileName := fmt.Sprintf("video_merge_test_%s.log", time.Now().Format("20060102_150405"))
	logFilePath := filepath.Join(logDir, logFileName)
	logFile, err := os.Create(logFilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	// 如果没有提供输入文件列表，则使用默认列表
	if len(inputFiles) == 0 {
		inputFiles = []string{"1.mp4", "2.mp4", "3.mp4", "4.mp4"}
	}

	return &VideoMergeTest{
		baseURL:    baseURL,
		logFile:    logFile,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logs:       make([]APILog, 0),
		startTime:  time.Now(),
		inputFiles: inputFiles, // 设置输入文件列表
	}, nil
}

// log 记录API调用日志
func (vmt *VideoMergeTest) log(method, url string, request, response interface{}, statusCode int, err error) {
	logEntry := APILog{
		Timestamp:  time.Now(),
		Method:     method,
		URL:        url,
		Request:    request,
		Response:   response,
		StatusCode: statusCode,
	}

	if err != nil {
		logEntry.Error = err.Error()
	}

	vmt.logs = append(vmt.logs, logEntry)

	// 写入日志文件
	logData, _ := json.MarshalIndent(logEntry, "", "  ")
	logData = append(logData, '\n')
	vmt.logFile.Write(logData)
	vmt.logFile.Sync()
}

// getVideoInfo 获取视频文件信息
func (vmt *VideoMergeTest) getVideoInfo(filePath string) (*VideoInfo, error) {
	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法获取文件信息: %w", err)
	}

	// 使用ffprobe获取视频信息
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe执行失败: %w", err)
	}

	// 解析JSON输出
	var probeData map[string]interface{}
	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("解析ffprobe输出失败: %w", err)
	}

	// 提取视频信息
	videoInfo := &VideoInfo{
		FileName: filePath,
		FileSize: fileInfo.Size(),
	}

	// 获取时长
	if format, ok := probeData["format"].(map[string]interface{}); ok {
		if durationStr, ok := format["duration"].(string); ok {
			fmt.Sscanf(durationStr, "%f", &videoInfo.Duration)
		}
		
		if bitRateStr, ok := format["bit_rate"].(string); ok {
			fmt.Sscanf(bitRateStr, "%d", &videoInfo.Bitrate)
		}
	}

	// 获取视频流信息
	if streams, ok := probeData["streams"].([]interface{}); ok {
		for _, stream := range streams {
			if streamMap, ok := stream.(map[string]interface{}); ok {
				if codecType, ok := streamMap["codec_type"].(string); ok && codecType == "video" {
					// 获取编码
					if codecName, ok := streamMap["codec_name"].(string); ok {
						videoInfo.Codec = codecName
					}
					
					// 获取尺寸
					if width, ok := streamMap["width"].(float64); ok {
						videoInfo.Width = int(width)
					}
					if height, ok := streamMap["height"].(float64); ok {
						videoInfo.Height = int(height)
					}
					
					// 获取FPS
					if avgFrameRate, ok := streamMap["avg_frame_rate"].(string); ok {
						var num, den int
						if _, err := fmt.Sscanf(avgFrameRate, "%d/%d", &num, &den); err == nil && den != 0 {
							videoInfo.FPS = float64(num) / float64(den)
						}
					}
					break
				}
			}
		}
	}

	return videoInfo, nil
}

// printVideoInfo 打印视频信息
func (vmt *VideoMergeTest) printVideoInfo(info *VideoInfo, label string) {
	fmt.Printf("\n%s:\n", label)
	fmt.Printf("  文件名: %s\n", info.FileName)
	fmt.Printf("  文件大小: %.2f MB\n", float64(info.FileSize)/(1024*1024))
	fmt.Printf("  时长: %.2f 秒\n", info.Duration)
	fmt.Printf("  编码: %s\n", info.Codec)
	fmt.Printf("  分辨率: %dx%d\n", info.Width, info.Height)
	fmt.Printf("  FPS: %.2f\n", info.FPS)
	fmt.Printf("  比特率: %d kbps\n", info.Bitrate/1000)
}

// healthCheck 健康检查
func (vmt *VideoMergeTest) healthCheck() error {
	url := vmt.baseURL + "/health"
	resp, err := vmt.httpClient.Get(url)
	if err != nil {
		vmt.log("GET", url, nil, nil, 0, err)
		return fmt.Errorf("failed to call health check: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	vmt.log("GET", url, nil, string(body), resp.StatusCode, nil)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check failed with status: %d", resp.StatusCode)
	}

	fmt.Println("Health check passed")
	return nil
}

// submitVideoEdit 提交视频编辑任务
func (vmt *VideoMergeTest) submitVideoEdit(request VideoEditRequest) (*VideoEditResponse, error) {
	url := vmt.baseURL + "/api/v1/video/edit"
	jsonData, _ := json.Marshal(request)

	fmt.Printf("提交视频编辑任务到: %s\n", url)
	fmt.Printf("请求数据: %s\n", string(jsonData))

	resp, err := vmt.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		vmt.log("POST", url, request, nil, 0, err)
		return nil, fmt.Errorf("failed to submit video edit task: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	vmt.log("POST", url, request, string(body), resp.StatusCode, nil)

	fmt.Printf("收到响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("响应数据: %s\n", string(body))

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("submit video edit task failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	var response VideoEditResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Printf("Task submitted successfully. Task ID: %s\n", response.TaskID)
	return &response, nil
}

// getTaskStatus 获取任务状态
func (vmt *VideoMergeTest) getTaskStatus(taskID string) (*TaskStatusResponse, error) {
	url := vmt.baseURL + "/api/v1/video/edit/" + taskID

	fmt.Printf("获取任务状态: %s\n", url)

	resp, err := vmt.httpClient.Get(url)
	if err != nil {
		vmt.log("GET", url, nil, nil, 0, err)
		return nil, fmt.Errorf("failed to get task status: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	vmt.log("GET", url, nil, string(body), resp.StatusCode, nil)

	fmt.Printf("任务状态响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("任务状态响应数据: %s\n", string(body))

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get task status failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	var response TaskStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// waitForTaskCompletion 等待任务完成
func (vmt *VideoMergeTest) waitForTaskCompletion(taskID string) (*TaskStatusResponse, error) {
	fmt.Printf("Waiting for task %s to complete...\n", taskID)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	// 显示处理进度
	lastProgress := -1.0
	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task completion")
		case <-ticker.C:
			status, err := vmt.getTaskStatus(taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to get task status: %w", err)
			}

			// 只有当进度发生变化时才打印
			if status.Progress != lastProgress {
				fmt.Printf("Task status: %s, Progress: %.2f%%\n", status.Status, status.Progress*100)
				lastProgress = status.Progress
			}

			switch status.Status {
			case "completed":
				fmt.Println("Task completed successfully")
				return status, nil
			case "failed":
				return status, fmt.Errorf("task failed: %s", status.Message)
			case "cancelled":
				return status, fmt.Errorf("task was cancelled")
			}
		}
	}
}

// cancelTask 取消任务
func (vmt *VideoMergeTest) cancelTask(taskID string) error {
	url := vmt.baseURL + "/api/v1/video/edit/" + taskID

	req, _ := http.NewRequest("DELETE", url, nil)
	resp, err := vmt.httpClient.Do(req)
	if err != nil {
		vmt.log("DELETE", url, nil, nil, 0, err)
		return fmt.Errorf("failed to cancel task: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	vmt.log("DELETE", url, nil, string(body), resp.StatusCode, nil)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("cancel task failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	fmt.Printf("Task %s cancelled successfully\n", taskID)
	return nil
}

// createVideoMergeSpec 创建视频合并规范
func (vmt *VideoMergeTest) createVideoMergeSpec() map[string]interface{} {
	// 构造符合worker期望的视频编辑规范
	// 生成简单的随机字符串替代UUID
	randomStr := fmt.Sprintf("%08x", time.Now().UnixNano()&0xFFFFFFFF)
	spec := map[string]interface{}{
		"inputs":  vmt.inputFiles, // 传递输入文件列表
		"outPath": fmt.Sprintf("./video/merged_output_%s.mp4", randomStr[:8]),
		"width":   1280,
		"height":  720,
		"fps":     30,
	}

	return spec
}

// printMaterialInfo 打印素材视频信息
func (vmt *VideoMergeTest) printMaterialInfo() error {
	fmt.Println("\n=== 素材视频信息 ===")
	
	// 使用配置的输入文件列表
	for _, material := range vmt.inputFiles {
		filePath := filepath.Join("video", material)
		info, err := vmt.getVideoInfo(filePath)
		if err != nil {
			fmt.Printf("获取 %s 信息失败: %v\n", material, err)
			continue
		}
		vmt.printVideoInfo(info, fmt.Sprintf("素材视频 %s", material))
	}
	
	return nil
}

// RunVideoMergeTest 运行视频合并测试
func (vmt *VideoMergeTest) RunVideoMergeTest() error {
	defer vmt.logFile.Close()
	
	// 记录开始时间
	vmt.startTime = time.Now()
	fmt.Printf("开始视频合并测试，时间: %s\n", vmt.startTime.Format("2006-01-02 15:04:05"))

	// 1. 打印素材视频信息
	if err := vmt.printMaterialInfo(); err != nil {
		return fmt.Errorf("打印素材信息失败: %w", err)
	}

	// 2. 健康检查
	if err := vmt.healthCheck(); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// 3. 创建视频合并规范
	spec := vmt.createVideoMergeSpec()
	
	// 4. 提交视频编辑任务
	request := VideoEditRequest{
		Spec: spec,
	}
	
	response, err := vmt.submitVideoEdit(request)
	if err != nil {
		return fmt.Errorf("failed to submit video edit task: %w", err)
	}

	// 5. 等待任务完成
	finalStatus, err := vmt.waitForTaskCompletion(response.TaskID)
	if err != nil {
		return fmt.Errorf("task did not complete successfully: %w", err)
	}

	// 6. 输出结果信息和耗时
	endTime := time.Now()
	duration := endTime.Sub(vmt.startTime)
	
	// 获取输出视频时长用于计算性能比率
	var outputDuration float64
	if finalStatus.OutputURL != "" {
		if info, err := vmt.getVideoInfo(finalStatus.OutputURL); err == nil {
			outputDuration = info.Duration
		}
	}
	
	fmt.Printf("\n=== 合成完成 ===\n")
	fmt.Printf("开始时间: %s\n", vmt.startTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("结束时间: %s\n", endTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("总耗时: %v\n", duration.Truncate(time.Millisecond))
	
	if outputDuration > 0 {
		ratio := float64(duration/time.Second) / outputDuration
		fmt.Printf("性能比率: %.2f (处理时间:视频时长)\n", ratio)
	}

	// 7. 打印输出视频信息
	if finalStatus.OutputURL != "" {
		info, err := vmt.getVideoInfo(finalStatus.OutputURL)
		if err != nil {
			fmt.Printf("获取输出视频信息失败: %v\n", err)
		} else {
			vmt.printVideoInfo(info, "合成视频")
		}
	}

	// 8. 验证输出文件是否存在
	if finalStatus.OutputURL != "" {
		if _, err := os.Stat(finalStatus.OutputURL); err == nil {
			fmt.Printf("\n输出文件存在且可访问\n")
		} else {
			fmt.Printf("\n警告: 输出文件未找到或不可访问: %v\n", err)
		}
	}

	// 打印日志文件路径
	logFilePath, _ := filepath.Abs(vmt.logFile.Name())
	fmt.Printf("\n日志文件: %s\n", logFilePath)
	return nil
}

// Run 运行视频合并测试的主函数
func Run() {
	// 创建测试实例，可以在这里修改输入文件列表
	inputFiles := []string{"1.ts", "2.ts", "3.ts", "4.ts", "5.ts"} // 修改这里的文件列表
	test, err := NewVideoMergeTest("http://localhost:8082", inputFiles)
	if err != nil {
		fmt.Printf("Failed to create test instance: %v\n", err)
		return
	}

	// 运行测试
	if err := test.RunVideoMergeTest(); err != nil {
		fmt.Printf("Test failed: %v\n", err)
	} else {
		fmt.Println("\n=== 测试完成 ===")
	}
}

func main() {
	Run()
}