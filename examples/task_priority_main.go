package example

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
)

// 模拟任务优先级处理
func handleTaskWithPriority(taskID string, priority int, spec interface{}) {
	fmt.Printf("处理任务 [%s] 优先级: %d\n", taskID, priority)
	
	// 模拟根据优先级调整处理时间
	baseDelay := time.Duration(1000/priority) * time.Millisecond
	time.Sleep(baseDelay)
	
	fmt.Printf("任务 [%s] 处理完成，耗时: %v\n", taskID, baseDelay)
}

// 模拟视频编辑API
func simulateVideoEditAPI(c *gin.Context) {
	// 解析请求
	var req struct {
		Spec     interface{} `json:"spec"`
		Priority int         `json:"priority"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效请求"})
		return
	}
	
	// 生成任务ID
	taskID := uuid.New().String()
	
	// 处理任务
	go handleTaskWithPriority(taskID, req.Priority, req.Spec)
	
	// 返回响应
	c.JSON(200, gin.H{
		"taskId":  taskID,
		"status":  "processing",
		"message": "任务已提交",
	})
}

func main() {
	// 创建测试服务器
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 注册路由
	router.POST("/video/edit", simulateVideoEditAPI)
	
	// 准备测试任务
	tasks := []struct {
		Spec     interface{}
		Priority int
	}{
		{map[string]interface{}{"type": "simple"}, 1},  // 低优先级
		{map[string]interface{}{"type": "complex"}, 5}, // 高优先级
		{map[string]interface{}{"type": "medium"}, 3},  // 中优先级
		{map[string]interface{}{"type": "urgent"}, 10}, // 紧急优先级
	}
	
	fmt.Println("=== 任务优先级处理测试 ===")
	
	// 发送测试请求
	for i, task := range tasks {
		// 构造请求
		requestBody := map[string]interface{}{
			"spec":     task.Spec,
			"priority": task.Priority,
		}
		
		jsonData, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/video/edit", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		// 创建响应记录器
		w := httptest.NewRecorder()
		
		// 执行请求
		fmt.Printf("\n发送任务 %d (优先级: %d)...\n", i+1, task.Priority)
		router.ServeHTTP(w, req)
		
		// 读取响应
		responseBody, _ := io.ReadAll(w.Body)
		fmt.Printf("响应状态: %d\n", w.Code)
		fmt.Printf("响应内容: %s\n", string(responseBody))
	}
	
	// 等待所有任务完成
	fmt.Println("\n等待所有任务完成...")
	time.Sleep(3 * time.Second)
	
	fmt.Println("\n=== 任务优先级测试完成 ===")
}

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