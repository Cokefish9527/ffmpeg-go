package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// VideoEditRequest 视频编辑请求
type VideoEditRequest struct {
	Spec     interface{} `json:"spec"`
	Priority int         `json:"priority,omitempty"`
}

// VideoEditResponse 视频编辑响应
type VideoEditResponse struct {
	TaskID   string `json:"taskId"`
	Status   string `json:"status"`
	Priority int    `json:"priority"`
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
	Priority  int     `json:"priority"`
}

func main() {
	fmt.Println("开始测试任务优先级功能...")
	
	// 提交不同优先级的任务
	task1 := submitTask("低优先级任务", 0) // PriorityLow
	task2 := submitTask("高优先级任务", 2) // PriorityHigh
	task3 := submitTask("普通优先级任务", 1) // PriorityNormal
	task4 := submitTask("紧急任务", 3) // PriorityCritical
	
	// 等待一段时间让任务开始处理
	time.Sleep(2 * time.Second)
	
	// 检查任务状态
	fmt.Println("\n=== 任务状态 ===")
	getTaskStatus(task1.TaskID)
	getTaskStatus(task2.TaskID)
	getTaskStatus(task3.TaskID)
	getTaskStatus(task4.TaskID)
	
	// 等待所有任务完成
	fmt.Println("\n=== 等待任务完成 ===")
	waitForTaskCompletion(task1.TaskID)
	waitForTaskCompletion(task2.TaskID)
	waitForTaskCompletion(task3.TaskID)
	waitForTaskCompletion(task4.TaskID)
}

func submitTask(name string, priority int) *VideoEditResponse {
	// 创建视频编辑规范
	spec := map[string]interface{}{
		"inputs": []string{
			"1.ts",
			"2.ts",
		},
		"outPath": fmt.Sprintf("./video/priority_test_%s.mp4", name),
		"width":   640,
		"height":  480,
		"fps":     30,
		"preset":  "ultrafast",
	}
	
	// 创建请求
	request := VideoEditRequest{
		Spec:     spec,
		Priority: priority,
	}
	
	// 发送请求
	jsonData, _ := json.Marshal(request)
	resp, err := http.Post("http://localhost:8082/api/v1/video/edit", "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("提交任务 %s 失败: %v\n", name, err)
		return nil
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusAccepted {
		fmt.Printf("提交任务 %s 失败，状态码: %d, 响应: %s\n", name, resp.StatusCode, string(body))
		return nil
	}
	
	var response VideoEditResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("解析任务 %s 响应失败: %v\n", name, err)
		return nil
	}
	
	fmt.Printf("提交任务 %s 成功，TaskID: %s，优先级: %d\n", name, response.TaskID, response.Priority)
	
	return &response
}

func getTaskStatus(taskID string) {
	resp, err := http.Get(fmt.Sprintf("http://localhost:8082/api/v1/video/edit/%s", taskID))
	if err != nil {
		fmt.Printf("获取任务 %s 状态失败: %v\n", taskID, err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := io.ReadAll(resp.Body)
	
	if resp.StatusCode != http.StatusOK {
		fmt.Printf("获取任务 %s 状态失败，状态码: %d, 响应: %s\n", taskID, resp.StatusCode, string(body))
		return
	}
	
	var response TaskStatusResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("解析任务 %s 状态响应失败: %v\n", taskID, err)
		return
	}
	
	fmt.Printf("任务 %s，状态: %s，优先级: %d\n", response.TaskID, response.Status, response.Priority)
}

func waitForTaskCompletion(taskID string) {
	for {
		resp, err := http.Get(fmt.Sprintf("http://localhost:8082/api/v1/video/edit/%s", taskID))
		if err != nil {
			fmt.Printf("获取任务 %s 状态失败: %v\n", taskID, err)
			return
		}
		defer resp.Body.Close()
		
		body, _ := io.ReadAll(resp.Body)
		
		if resp.StatusCode != http.StatusOK {
			fmt.Printf("获取任务 %s 状态失败，状态码: %d, 响应: %s\n", taskID, resp.StatusCode, string(body))
			return
		}
		
		var response TaskStatusResponse
		if err := json.Unmarshal(body, &response); err != nil {
			fmt.Printf("解析任务 %s 状态响应失败: %v\n", taskID, err)
			return
		}
		
		if response.Status == "completed" || response.Status == "failed" {
			fmt.Printf("任务 %s 已完成，状态: %s\n", response.TaskID, response.Status)
			return
		}
		
		fmt.Printf("任务 %s 正在处理中，状态: %s\n", response.TaskID, response.Status)
		time.Sleep(1 * time.Second)
	}
}