package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
)

// setupIntegrationTestServer 设置集成测试服务器
func setupIntegrationTestServer() (*gin.Engine, queue.TaskQueue, *service.WorkerPool) {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 初始化任务队列
	taskQueue := queue.NewInMemoryTaskQueue()

	// 初始化视频编辑服务
	editorService := service.NewVideoEditorService(taskQueue)

	// 初始化工作池，使用2个worker
	workerPool := service.NewWorkerPool(2, taskQueue)

	// 创建Gin引擎
	r := gin.New()
	r.Use(gin.Recovery())

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// 定义API路由组
	apiRoutes := r.Group("/api/v1")
	{
		apiRoutes.POST("/video/edit", func(c *gin.Context) {
			var req service.VideoEditRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid request format",
				})
				return
			}

			// 提交任务到视频编辑服务
			task, err := editorService.SubmitTask(&req)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to submit task",
				})
				return
			}

			// 返回成功响应
			response := api.VideoEditResponse{
				TaskID:  task.ID,
				Status:  task.Status,
				Message: "Task accepted for processing",
			}

			c.JSON(http.StatusAccepted, response)
		})

		apiRoutes.GET("/video/edit/:taskId", func(c *gin.Context) {
			taskID := c.Param("taskId")

			task, err := editorService.GetTaskStatus(taskID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to get task status",
				})
				return
			}

			if task == nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Task not found",
				})
				return
			}

			response := api.TaskStatusResponse{
				TaskID:   task.ID,
				Status:   task.Status,
				Progress: task.Progress,
				Message:  task.Error,
				Created:  task.Created.Format(time.RFC3339),
			}

			if !task.Started.IsZero() {
				response.Started = task.Started.Format(time.RFC3339)
			}

			if !task.Finished.IsZero() {
				response.Finished = task.Finished.Format(time.RFC3339)
				response.OutputURL = task.Result
			}

			c.JSON(http.StatusOK, response)
		})

		apiRoutes.DELETE("/video/edit/:taskId", func(c *gin.Context) {
			taskID := c.Param("taskId")

			err := editorService.CancelTask(taskID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":  "Failed to cancel task",
					"taskId": taskID,
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Task cancelled successfully",
				"taskId":  taskID,
			})
		})
		
		// 添加监控API路由
		monitorAPI := api.NewMonitorAPI(taskQueue, workerPool)
		apiRoutes.GET("/monitor/stats", monitorAPI.GetSystemStats)
		apiRoutes.GET("/monitor/tasks/stats", monitorAPI.GetTaskStats)
		apiRoutes.GET("/monitor/tasks", monitorAPI.GetTasks)
		apiRoutes.GET("/monitor/tasks/:taskId", monitorAPI.GetTaskDetail)
		apiRoutes.GET("/monitor/workers", monitorAPI.GetWorkerStats)
		// 添加任务管理接口
		apiRoutes.POST("/monitor/tasks/retry", monitorAPI.RetryTask)
		apiRoutes.POST("/monitor/tasks/cancel", monitorAPI.CancelTask)
	}

	// 启动工作池
	workerPool.Start()

	return r, taskQueue, workerPool
}

// TestAPISuite 运行完整的API测试套件
func TestAPISuite(t *testing.T) {
	// 检查示例数据文件是否存在
	if _, err := os.Stat("../examples/sample_data/in1.mp4"); os.IsNotExist(err) {
		t.Skip("Skipping integration test: sample data not found")
	}

	router, taskQueue, workerPool := setupIntegrationTestServer()
	defer workerPool.Stop()

	// 用于存储任务ID，供后续测试使用
	var taskID string

	// 测试1: 健康检查
	t.Run("HealthCheck", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "ok", response["status"])
	})

	// 测试2: 提交视频编辑任务
	t.Run("SubmitVideoEdit", func(t *testing.T) {
		// 构造请求数据
		requestData := service.VideoEditRequest{
			Spec: map[string]interface{}{
				"inputs": []map[string]interface{}{
					{
						"source": "../examples/sample_data/in1.mp4",
					},
				},
				"output": map[string]interface{}{
					"filename": "../examples/sample_data/output_test.mp4",
				},
			},
		}

		jsonData, _ := json.Marshal(requestData)
		req, _ := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusAccepted, w.Code)

		var response api.VideoEditResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response.TaskID)
		assert.Equal(t, "pending", response.Status)
		assert.Equal(t, "Task accepted for processing", response.Message)

		// 保存任务ID供后续测试使用
		taskID = response.TaskID
	})

	// 测试3: 获取任务状态
	t.Run("GetTaskStatus", func(t *testing.T) {
		if taskID == "" {
			t.Skip("Skipping: no task ID from previous test")
		}

		// 查询任务状态
		req, _ := http.NewRequest("GET", "/api/v1/video/edit/"+taskID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.TaskStatusResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, taskID, response.TaskID)
		// 任务应该在处理队列中
		assert.Contains(t, []string{"pending", "processing", "completed", "failed"}, response.Status)
	})

	// 测试4: 等待任务完成并验证最终状态
	t.Run("WaitForTaskCompletion", func(t *testing.T) {
		if taskID == "" {
			t.Skip("Skipping: no task ID from previous test")
		}

		// 等待任务处理完成
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		taskCompleted := make(chan bool)
		go func() {
			for {
				task, _ := taskQueue.Get(taskID)
				if task != nil && (task.Status == "completed" || task.Status == "failed") {
					taskCompleted <- true
					return
				}
				select {
				case <-ctx.Done():
					taskCompleted <- false
					return
				case <-time.After(200 * time.Millisecond):
					// 继续检查
				}
			}
		}()

		// 等待任务完成或超时
		completed := false
		select {
		case completed = <-taskCompleted:
			// 任务完成
		case <-ctx.Done():
			// 超时
		}

		assert.True(t, completed, "Task should be completed within timeout")

		// 验证最终状态
		task, err := taskQueue.Get(taskID)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Contains(t, []string{"completed", "failed"}, task.Status)
	})

	// 测试5: 再次获取任务状态
	t.Run("GetFinalTaskStatus", func(t *testing.T) {
		if taskID == "" {
			t.Skip("Skipping: no task ID from previous test")
		}

		// 查询任务状态
		req, _ := http.NewRequest("GET", "/api/v1/video/edit/"+taskID, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response api.TaskStatusResponse
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, taskID, response.TaskID)
		// 任务应该已完成或失败
		assert.Contains(t, []string{"completed", "failed"}, response.Status)

		// 如果任务已完成，应该有完成时间
		if response.Status == "completed" {
			assert.NotEmpty(t, response.Finished)
		}
	})

	// 测试6: 提交一个无效的任务
	t.Run("SubmitInvalidTask", func(t *testing.T) {
		// 发送无效的JSON数据
		req, _ := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer([]byte("{ invalid json }")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Invalid request format", response["error"])
	})

	// 测试7: 查询不存在的任务
	t.Run("GetNonExistentTask", func(t *testing.T) {
		// 查询一个不存在的任务
		req, _ := http.NewRequest("GET", "/api/v1/video/edit/nonexistent-task-id", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)

		var response map[string]string
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Task not found", response["error"])
	})
	
	// 测试8: 监控API - 获取任务统计信息
	t.Run("GetTaskStats", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/monitor/tasks/stats", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Contains(t, response, "totalTasks")
		assert.Contains(t, response, "pendingTasks")
		assert.Contains(t, response, "processingTasks")
		assert.Contains(t, response, "completedTasks")
		assert.Contains(t, response, "failedTasks")
	})
	
	// 测试9: 监控API - 获取任务列表
	t.Run("GetTasks", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/api/v1/monitor/tasks", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response []map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.NotEmpty(t, response)
	})
	
	// 测试10: 监控API - 任务重试功能
	t.Run("RetryTask", func(t *testing.T) {
		if taskID == "" {
			t.Skip("Skipping: no task ID from previous test")
		}
		
		// 首先确认任务是失败状态
		task, err := taskQueue.Get(taskID)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "failed", task.Status)
		
		// 调用重试接口
		retryRequest := map[string]string{
			"taskId": taskID,
		}
		jsonData, _ := json.Marshal(retryRequest)
		req, _ := http.NewRequest("POST", "/api/v1/monitor/tasks/retry", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Task retry successfully", response["message"])
		
		// 验证任务状态是否已更新为pending
		task, err = taskQueue.Get(taskID)
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "pending", task.Status)
		assert.Empty(t, task.Error)
	})
	
	// 测试11: 监控API - 任务丢弃功能
	t.Run("DiscardTask", func(t *testing.T) {
		// 创建一个新任务用于测试丢弃功能
		task := &queue.Task{
			ID:       "test-discard-task",
			Status:   "failed",
			Spec:     map[string]interface{}{"test": "data"},
			Error:    "test error",
			Priority: queue.PriorityNormal,
			Created:  time.Now(),
		}
		
		err := taskQueue.Update(task)
		assert.NoError(t, err)
		
		// 验证任务创建成功
		createdTask, err := taskQueue.Get("test-discard-task")
		assert.NoError(t, err)
		assert.NotNil(t, createdTask)
		assert.Equal(t, "failed", createdTask.Status)
		
		// 调用丢弃接口
		discardRequest := map[string]string{
			"taskId": "test-discard-task",
		}
		jsonData, _ := json.Marshal(discardRequest)
		req, _ := http.NewRequest("POST", "/api/v1/monitor/tasks/discard", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		
		// 打印响应内容用于调试
		t.Logf("DiscardTask response status: %d", w.Code)
		t.Logf("DiscardTask response body: %s", w.Body.String())
		
		assert.Equal(t, http.StatusOK, w.Code)
		
		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		assert.NoError(t, err)
		assert.Equal(t, "Task discarded successfully", response["message"])
		
		// 验证任务状态是否已更新为discarded
		task, err = taskQueue.Get("test-discard-task")
		assert.NoError(t, err)
		assert.NotNil(t, task)
		assert.Equal(t, "discarded", task.Status)
	})
}