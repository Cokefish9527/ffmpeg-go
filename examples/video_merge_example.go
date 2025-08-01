package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/google/uuid"
)

// VideoEditSpec 已移除，使用interface{}代替

// OutputSpec 输出规范
type OutputSpec struct {
	Filename string `json:"filename"`
}

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

// VideoMergeTest 视频合并测试结构
type VideoMergeTest struct {
	baseURL    string
	logFile    *os.File
	httpClient *http.Client
	logs       []APILog
}

// NewVideoMergeTest 创建新的视频合并测试实例
func NewVideoMergeTest(baseURL string) (*VideoMergeTest, error) {
	// 创建日志文件
	logFileName := fmt.Sprintf("video_merge_test_%s.log", time.Now().Format("20060102_150405"))
	logFile, err := os.Create(logFileName)
	if err != nil {
		return nil, fmt.Errorf("failed to create log file: %w", err)
	}

	return &VideoMergeTest{
		baseURL:    baseURL,
		logFile:    logFile,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		logs:       make([]APILog, 0),
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

	resp, err := vmt.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		vmt.log("POST", url, request, nil, 0, err)
		return nil, fmt.Errorf("failed to submit video edit task: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	vmt.log("POST", url, request, string(body), resp.StatusCode, nil)

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

	resp, err := vmt.httpClient.Get(url)
	if err != nil {
		vmt.log("GET", url, nil, nil, 0, err)
		return nil, fmt.Errorf("failed to get task status: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	vmt.log("GET", url, nil, string(body), resp.StatusCode, nil)

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

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task completion")
		case <-ticker.C:
			status, err := vmt.getTaskStatus(taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to get task status: %w", err)
			}

			fmt.Printf("Task status: %s, Progress: %.2f%%\n", status.Status, status.Progress*100)

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
	spec := map[string]interface{}{
		"outPath": fmt.Sprintf("./video/merged_output_%s.mp4", uuid.New().String()[:8]),
		"width":   1280,
		"height":  720,
		"fps":     30,
	}

	return spec
}

// RunVideoMergeTest 运行视频合并测试
func (vmt *VideoMergeTest) RunVideoMergeTest() error {
	defer vmt.logFile.Close()

	fmt.Println("Starting video merge test...")

	// 1. 健康检查
	if err := vmt.healthCheck(); err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}

	// 2. 创建视频合并规范
	spec := vmt.createVideoMergeSpec()
	
	// 3. 提交视频编辑任务
	request := VideoEditRequest{
		Spec: spec,
	}
	
	response, err := vmt.submitVideoEdit(request)
	if err != nil {
		return fmt.Errorf("failed to submit video edit task: %w", err)
	}

	// 4. 等待任务完成
	finalStatus, err := vmt.waitForTaskCompletion(response.TaskID)
	if err != nil {
		return fmt.Errorf("task did not complete successfully: %w", err)
	}

	// 5. 输出结果信息
	fmt.Printf("Video merge completed!\n")
	fmt.Printf("Output file: %s\n", finalStatus.OutputURL)
	fmt.Printf("Log file: %s\n", vmt.logFile.Name())

	// 6. 验证输出文件是否存在
	if finalStatus.OutputURL != "" {
		if _, err := os.Stat(finalStatus.OutputURL); err == nil {
			fmt.Printf("Output file exists and is accessible\n")
		} else {
			fmt.Printf("Warning: Output file not found or not accessible: %v\n", err)
		}
	}

	return nil
}

func main() {
	// 创建测试实例
	test, err := NewVideoMergeTest("http://localhost:8082")
	if err != nil {
		fmt.Printf("Failed to create test instance: %v\n", err)
		return
	}

	// 运行测试
	if err := test.RunVideoMergeTest(); err != nil {
		fmt.Printf("Test failed: %v\n", err)
	} else {
		fmt.Println("Test completed successfully")
	}
}