package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// ConcurrentTest 并发测试结构
type ConcurrentTest struct {
	baseURL    string
	httpClient *http.Client
}

// NewConcurrentTest 创建新的并发测试实例
func NewConcurrentTest(baseURL string) *ConcurrentTest {
	return &ConcurrentTest{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 60 * time.Second},
	}
}

// VideoEditRequest 视频编辑请求
type VideoEditRequest struct {
	Spec interface{} `json:"spec"`
}

// VideoEditResponse 视频编辑响应
type VideoEditResponse struct {
	TaskID string `json:"taskId"`
	Status string `json:"status"`
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

// submitVideoEdit 提交视频编辑任务
func (ct *ConcurrentTest) submitVideoEdit(request VideoEditRequest) (*VideoEditResponse, error) {
	url := ct.baseURL + "/api/v1/video/edit"
	jsonData, _ := json.Marshal(request)

	resp, err := ct.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to submit video edit task: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusAccepted {
		return nil, fmt.Errorf("submit video edit task failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	var response VideoEditResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &response, nil
}

// getTaskStatus 获取任务状态
func (ct *ConcurrentTest) getTaskStatus(taskID string) (*TaskStatusResponse, error) {
	url := ct.baseURL + "/api/v1/video/edit/" + taskID

	resp, err := ct.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get task status: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

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
func (ct *ConcurrentTest) waitForTaskCompletion(taskID string) (*TaskStatusResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil, fmt.Errorf("timeout waiting for task completion")
		case <-ticker.C:
			status, err := ct.getTaskStatus(taskID)
			if err != nil {
				return nil, fmt.Errorf("failed to get task status: %w", err)
			}

			switch status.Status {
			case "completed":
				return status, nil
			case "failed":
				return status, fmt.Errorf("task failed: %s", status.Message)
			case "cancelled":
				return status, fmt.Errorf("task was cancelled")
			}
		}
	}
}

// createVideoMergeSpec 创建视频合并规范
func (ct *ConcurrentTest) createVideoMergeSpec(index int) map[string]interface{} {
	spec := map[string]interface{}{
		"inputs": []string{
			"1.ts",
			"2.ts",
			"3.ts",
			"4.ts",
			"5.ts",
		},
		"outPath": fmt.Sprintf("./video/concurrent_output_%d.mp4", index),
		"width":   1280,
		"height":  720,
		"fps":     30,
		"preset":  "ultrafast",
	}

	return spec
}

// getWorkerPoolStatus 获取WorkerPool状态
func (ct *ConcurrentTest) getWorkerPoolStatus() (map[string]interface{}, error) {
	url := ct.baseURL + "/api/v1/workerpool/status"

	resp, err := ct.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to get worker pool status: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get worker pool status failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	var status map[string]interface{}
	if err := json.Unmarshal(body, &status); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return status, nil
}

// resizeWorkerPool 调整WorkerPool大小
func (ct *ConcurrentTest) resizeWorkerPool(size int) error {
	url := ct.baseURL + "/api/v1/workerpool/resize"
	
	request := map[string]interface{}{
		"size": size,
	}
	
	jsonData, _ := json.Marshal(request)

	resp, err := ct.httpClient.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to resize worker pool: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("resize worker pool failed with status: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// runConcurrentTest 运行并发测试
func (ct *ConcurrentTest) runConcurrentTest(concurrentTasks int) error {
	fmt.Printf("开始并发测试，同时处理 %d 个任务\n", concurrentTasks)
	
	startTime := time.Now()
	
	// 获取初始WorkerPool状态
	initialStatus, err := ct.getWorkerPoolStatus()
	if err != nil {
		return fmt.Errorf("failed to get initial worker pool status: %w", err)
	}
	
	fmt.Printf("初始WorkerPool状态: %+v\n", initialStatus)
	
	// 调整WorkerPool大小以适应并发任务
	if concurrentTasks > int(initialStatus["workerCount"].(float64)) {
		fmt.Printf("调整WorkerPool大小到 %d\n", concurrentTasks)
		if err := ct.resizeWorkerPool(concurrentTasks); err != nil {
			return fmt.Errorf("failed to resize worker pool: %w", err)
		}
		
		// 等待调整生效
		time.Sleep(1 * time.Second)
		
		// 获取调整后的WorkerPool状态
		resizedStatus, err := ct.getWorkerPoolStatus()
		if err != nil {
			return fmt.Errorf("failed to get resized worker pool status: %w", err)
		}
		
		fmt.Printf("调整后WorkerPool状态: %+v\n", resizedStatus)
	}
	
	// 创建等待组
	var wg sync.WaitGroup
	results := make(chan struct {
		taskIndex int
		duration  time.Duration
		err       error
	}, concurrentTasks)
	
	// 启动并发任务
	for i := 0; i < concurrentTasks; i++ {
		wg.Add(1)
		go func(taskIndex int) {
			defer wg.Done()
			
			taskStartTime := time.Now()
			
			// 创建视频合并规范
			spec := ct.createVideoMergeSpec(taskIndex)
			
			// 提交视频编辑任务
			request := VideoEditRequest{
				Spec: spec,
			}
			
			response, err := ct.submitVideoEdit(request)
			if err != nil {
				results <- struct {
					taskIndex int
					duration  time.Duration
					err       error
				}{taskIndex, 0, fmt.Errorf("任务 %d 提交失败: %w", taskIndex, err)}
				return
			}
			
			fmt.Printf("任务 %d 已提交，TaskID: %s\n", taskIndex, response.TaskID)
			
			// 等待任务完成
			_, err = ct.waitForTaskCompletion(response.TaskID)
			duration := time.Since(taskStartTime)
			
			if err != nil {
				results <- struct {
					taskIndex int
					duration  time.Duration
					err       error
				}{taskIndex, duration, fmt.Errorf("任务 %d 执行失败: %w", taskIndex, err)}
				return
			}
			
			results <- struct {
				taskIndex int
				duration  time.Duration
				err       error
			}{taskIndex, duration, nil}
		}(i)
	}
	
	// 等待所有任务完成
	go func() {
		wg.Wait()
		close(results)
	}()
	
	// 收集结果
	completedTasks := 0
	failedTasks := 0
	totalDuration := time.Duration(0)
	
	for result := range results {
		if result.err != nil {
			fmt.Printf("任务 %d 失败: %v\n", result.taskIndex, result.err)
			failedTasks++
		} else {
			fmt.Printf("任务 %d 完成，耗时: %v\n", result.taskIndex, result.duration)
			completedTasks++
			totalDuration += result.duration
		}
	}
	
	totalTime := time.Since(startTime)
	
	fmt.Printf("\n=== 并发测试结果 ===\n")
	fmt.Printf("并发任务数: %d\n", concurrentTasks)
	fmt.Printf("成功完成: %d\n", completedTasks)
	fmt.Printf("失败: %d\n", failedTasks)
	fmt.Printf("总耗时: %v\n", totalTime)
	
	if completedTasks > 0 {
		avgDuration := totalDuration / time.Duration(completedTasks)
		fmt.Printf("平均任务耗时: %v\n", avgDuration)
		fmt.Printf("吞吐量: %.2f 任务/秒\n", float64(completedTasks)/totalTime.Seconds())
	}
	
	return nil
}

func main() {
    // 确认 baseURL 正确指向服务的实际地址
    test := NewConcurrentTest("http://127.0.0.1:8082") // 使用 127.0.0.1 替换 localhost
    
    // 运行并发测试，同时处理3个任务
    if err := test.runConcurrentTest(3); err != nil {
        fmt.Printf("并发测试失败: %v\n", err)
    }
}