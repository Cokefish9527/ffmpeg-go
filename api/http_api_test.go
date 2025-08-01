package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/u2takey/ffmpeg-go/service"
)

// setupTestServer 设置测试服务器
func setupTestServer() (*gin.Engine, *service.InMemoryTaskQueue, *service.WorkerPool) {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 初始化任务队列
	taskQueue := service.NewInMemoryTaskQueue()

	// 初始化工作池，使用1个worker以避免测试过于复杂
	workerPool := service.NewWorkerPool(1)

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
			var req VideoEditRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid request format",
				})
				return
			}

			// 生成任务ID
			taskID := uuid.New().String()

			// 创建任务对象
			task := &service.Task{
				ID:       taskID,
				Spec:     req.Spec,
				Status:   "pending",
				Created:  time.Now(),
				Progress: 0.0,
			}

			// 将任务添加到队列
			if err := taskQueue.Add(task); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to add task to queue",
				})
				return
			}

			// 提交任务到工作池
			if err := workerPool.SubmitTask(task); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to submit task to worker pool",
				})
				return
			}

			// 返回成功响应
			response := VideoEditResponse{
				TaskID:  taskID,
				Status:  "accepted",
				Message: "Task accepted for processing",
			}

			c.JSON(http.StatusAccepted, response)
		})

		apiRoutes.GET("/video/edit/:taskId", func(c *gin.Context) {
			taskID := c.Param("taskId")

			task, err := taskQueue.Get(taskID)
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

			response := TaskStatusResponse{
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

			// 获取任务
			task, err := taskQueue.Get(taskID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error":  "Failed to get task",
					"taskId": taskID,
				})
				return
			}

			if task == nil {
				c.JSON(http.StatusNotFound, gin.H{
					"error":  "Task not found",
					"taskId": taskID,
				})
				return
			}

			// 只有在任务未完成时才能取消
			if task.Status == "pending" || task.Status == "processing" {
				task.Status = "cancelled"
				task.Error = "任务已被取消"
				// 更新任务状态
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Task cancelled successfully",
				"taskId":  taskID,
			})
		})
	}

	// 启动工作池
	workerPool.Start()

	return r, taskQueue, workerPool
}

// TestHealthCheck 测试健康检查接口
func TestHealthCheck(t *testing.T) {
	router, _, _ := setupTestServer()

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "ok", response["status"])
}

// TestSubmitVideoEdit 测试提交视频编辑任务接口
func TestSubmitVideoEdit(t *testing.T) {
	router, _, _ := setupTestServer()

	// 构造请求数据
	requestData := VideoEditRequest{
		Spec: map[string]interface{}{
			"inputs": []map[string]interface{}{
				{
					"source": "./examples/sample_data/in1.mp4",
				},
			},
			"output": map[string]interface{}{
				"filename": "./examples/sample_data/output.mp4",
			},
		},
		OutputPath: "./examples/sample_data/output.mp4",
	}

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var response VideoEditResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.TaskID)
	assert.Equal(t, "accepted", response.Status)
	assert.Equal(t, "Task accepted for processing", response.Message)
}

// TestSubmitVideoEditInvalidJSON 测试提交无效JSON数据
func TestSubmitVideoEditInvalidJSON(t *testing.T) {
	router, _, _ := setupTestServer()

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
}

// TestGetVideoEditStatus 测试获取视频编辑任务状态接口
func TestGetVideoEditStatus(t *testing.T) {
	router, taskQueue, _ := setupTestServer()

	// 先创建一个任务
	task := &service.Task{
		ID:       uuid.New().String(),
		Spec:     map[string]interface{}{},
		Status:   "pending",
		Created:  time.Now(),
		Progress: 0.0,
	}
	taskQueue.Add(task)

	// 查询任务状态
	req, _ := http.NewRequest("GET", "/api/v1/video/edit/"+task.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response TaskStatusResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, task.ID, response.TaskID)
	assert.Equal(t, task.Status, response.Status)
	assert.Equal(t, task.Progress, response.Progress)
}

// TestGetVideoEditStatusNotFound 测试获取不存在的任务状态
func TestGetVideoEditStatusNotFound(t *testing.T) {
	router, _, _ := setupTestServer()

	// 查询一个不存在的任务
	req, _ := http.NewRequest("GET", "/api/v1/video/edit/nonexistent-task-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Task not found", response["error"])
}

// TestCancelVideoEdit 测试取消视频编辑任务接口
func TestCancelVideoEdit(t *testing.T) {
	router, taskQueue, _ := setupTestServer()

	// 先创建一个任务
	task := &service.Task{
		ID:       uuid.New().String(),
		Spec:     map[string]interface{}{},
		Status:   "pending",
		Created:  time.Now(),
		Progress: 0.0,
	}
	taskQueue.Add(task)

	// 取消任务
	req, _ := http.NewRequest("DELETE", "/api/v1/video/edit/"+task.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Task cancelled successfully", response["message"])
	assert.Equal(t, task.ID, response["taskId"])
}

// TestCancelVideoEditNotFound 测试取消不存在的视频编辑任务
func TestCancelVideoEditNotFound(t *testing.T) {
	router, _, _ := setupTestServer()

	// 取消一个不存在的任务
	req, _ := http.NewRequest("DELETE", "/api/v1/video/edit/nonexistent-task-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, "Task not found", response["error"])
}

// TestFullWorkflow 测试完整工作流程
func TestFullWorkflow(t *testing.T) {
	router, taskQueue, workerPool := setupTestServer()
	defer workerPool.Stop()

	// 1. 提交任务
	requestData := VideoEditRequest{
		Spec: map[string]interface{}{
			"inputs": []map[string]interface{}{
				{
					"source": "./examples/sample_data/in1.mp4",
				},
			},
			"output": map[string]interface{}{
				"filename": "./examples/sample_data/output.mp4",
			},
		},
		OutputPath: "./examples/sample_data/output.mp4",
	}

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusAccepted, w.Code)

	var submitResponse VideoEditResponse
	err := json.Unmarshal(w.Body.Bytes(), &submitResponse)
	assert.NoError(t, err)
	taskID := submitResponse.TaskID

	// 2. 检查任务状态（立即检查）
	req, _ = http.NewRequest("GET", "/api/v1/video/edit/"+taskID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var statusResponse TaskStatusResponse
	err = json.Unmarshal(w.Body.Bytes(), &statusResponse)
	assert.NoError(t, err)
	assert.Equal(t, taskID, statusResponse.TaskID)
	// 任务应该在处理队列中（可能是pending或processing状态，取决于处理速度）
	assert.Contains(t, []string{"pending", "processing"}, statusResponse.Status)

	// 3. 等待任务处理完成
	// 等待一段时间让worker处理任务
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
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
			case <-time.After(100 * time.Millisecond):
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

	// 4. 再次检查任务状态
	req, _ = http.NewRequest("GET", "/api/v1/video/edit/"+taskID, nil)
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	err = json.Unmarshal(w.Body.Bytes(), &statusResponse)
	assert.NoError(t, err)
	assert.Equal(t, taskID, statusResponse.TaskID)
	// 任务应该已完成或失败（根据worker的实现）
	assert.Contains(t, []string{"completed", "failed"}, statusResponse.Status)
}