package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// VideoEditRequest 视频编辑请求
type VideoEditRequest struct {
	Spec interface{} `json:"spec"`
}

// VideoEditResponse 视频编辑响应
type VideoEditResponse struct {
	TaskID  string `json:"taskId"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

func main() {
	// 准备测试数据
	editSpec := map[string]interface{}{
		"outPath": "./test_output.mp4",
		"width":   1920,
		"height":  1080,
		"fps":     30,
		"clips": []map[string]interface{}{
			{
				"duration": 5,
				"layers": []map[string]interface{}{
					{
						"type": "video",
						"path": "http://example.com/test1.mp4",
					},
				},
			},
			{
				"duration": 5,
				"layers": []map[string]interface{}{
					{
						"type": "video",
						"path": "http://example.com/test2.mp4",
					},
				},
			},
		},
	}

	requestData := VideoEditRequest{
		Spec: editSpec,
	}

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("Error marshaling JSON: %v\n", err)
		return
	}

	// 创建HTTP请求
	fmt.Println("Sending request to video/edit endpoint...")
	startTime := time.Now()
	
	resp, err := http.Post("http://localhost:8082/api/v1/video/edit", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("Error sending request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	elapsedTime := time.Since(startTime)
	fmt.Printf("Request completed in %v\n", elapsedTime)

	// 检查响应状态
	if resp.StatusCode != http.StatusAccepted {
		fmt.Printf("Expected status %d, got %d\n", http.StatusAccepted, resp.StatusCode)
		return
	}

	// 解析响应数据
	var response VideoEditResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		fmt.Printf("Error decoding response: %v\n", err)
		return
	}

	// 输出响应内容
	fmt.Printf("Response:\n")
	fmt.Printf("  TaskID: %s\n", response.TaskID)
	fmt.Printf("  Status: %s\n", response.Status)
	fmt.Printf("  Message: %s\n", response.Message)
}