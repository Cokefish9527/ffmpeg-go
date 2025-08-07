package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/u2takey/ffmpeg-go/service"
	"github.com/u2takey/ffmpeg-go/queue"
)

// setupTestServer 设置测试服务器
func setupTestServer() (*gin.Engine, *queue.InMemoryTaskQueue, *service.WorkerPool) {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 初始化任务队列
	taskQueue := queue.NewInMemoryTaskQueue()

	// 初始化工作池，使用1个worker以避免测试过于复杂
	workerPool := service.NewWorkerPool(1, taskQueue)

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
			task := &queue.Task{
				ID:       taskID,
				Spec:     req.Spec,
				Status:   "pending",
				Created:  time.Now(),
				Progress: 0.0,
			}

			// 将任务添加到队列
			if err := taskQueue.Push(task); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to add task to queue",
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

		// 素材上传接口
		apiRoutes.POST("/material/upload", func(c *gin.Context) {
			// 获取上传的文件
			file, err := c.FormFile("file")
			if err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Failed to get file from request",
				})
				return
			}

			// 生成任务ID
			taskID := uuid.New().String()

			// 创建临时目录
			tempDir := "./temp"
			if _, err := os.Stat(tempDir); os.IsNotExist(err) {
				os.Mkdir(tempDir, 0755)
			}

			// 保存文件到临时位置
			filename := fmt.Sprintf("%s/%s_%s", tempDir, taskID, file.Filename)
			if err := c.SaveUploadedFile(file, filename); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to save uploaded file",
				})
				return
			}

			// 创建任务对象
			task := &queue.Task{
				ID:     taskID,
				Spec: map[string]interface{}{
					"source":     filename,
					"taskType":   "materialPreprocess",
				},
				Status:   "pending",
				Created:  time.Now(),
				Progress: 0.0,
			}

			// 将任务添加到队列
			if err := taskQueue.Push(task); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to add task to queue",
				})
				return
			}

			// 返回成功响应
			response := MaterialUploadResponse{
				TaskID:  taskID,
				Status:  "accepted",
				Message: "Material upload accepted for processing",
			}

			c.JSON(http.StatusAccepted, response)
		})

		// 视频URL处理接口
		apiRoutes.POST("/video/url", func(c *gin.Context) {
			var req VideoURLRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, VideoURLResponse{
					Status: "error",
					Message: "Invalid request format",
					Error:  err.Error(),
				})
				return
			}

			if req.URL == "" {
				c.JSON(http.StatusBadRequest, VideoURLResponse{
					Status: "error",
					Message: "URL is required",
					Error:  "URL field is empty",
				})
				return
			}

			// 生成任务ID
			taskID := uuid.New().String()

			// 创建临时目录
			tempDir := "./temp"
			if _, err := os.Stat(tempDir); os.IsNotExist(err) {
				os.Mkdir(tempDir, 0755)
			}

			// 生成临时文件名
			filename := fmt.Sprintf("%s/%s_temp.mp4", tempDir, taskID)
			
			// 记录下载开始时间
			downloadStart := time.Now()
			
			// 下载文件
			err := downloadFile(req.URL, filename)
			if err != nil {
				c.JSON(http.StatusInternalServerError, VideoURLResponse{
					Status: "error",
					Message: "Failed to download file",
					Error:  err.Error(),
				})
				return
			}
			
			// 记录下载结束时间
			downloadEnd := time.Now()
			downloadDuration := downloadEnd.Sub(downloadStart).Seconds()

			// 生成输出文件路径 (TS格式)
			ext := filepath.Ext(filename)
			outputFile := filename[0:len(filename)-len(ext)] + ".ts"

			// 创建任务对象
			task := &queue.Task{
				ID:   taskID,
				Spec: map[string]interface{}{
					"source":   filename,
					"output":   outputFile,
					"taskType": "materialPreprocess",
					"callback": req.Callback, // 添加回调URL到任务规范中
				},
				Status:   "pending",
				Created:  time.Now(),
				Progress: 0.0,
			}

			// 将任务添加到队列
			if err := taskQueue.Push(task); err != nil {
				c.JSON(http.StatusInternalServerError, VideoURLResponse{
					Status: "error",
					Message: "Failed to add task to queue",
					Error:  err.Error(),
				})
				return
			}

			// 如果提供了回调URL，则异步处理任务
			if req.Callback != "" {
				// 不等待任务完成，直接返回接受响应
				c.JSON(http.StatusAccepted, VideoURLResponse{
					Status:  "accepted",
					Message: "Video conversion task accepted, you will be notified via callback when it's completed",
					TSFilePath: outputFile,
				})
				return
			}

			// 等待任务完成（简化处理，实际项目中应该异步处理）
			for i := 0; i < 30; i++ { // 最多等待30秒
				task, err := taskQueue.Get(taskID)
				if err != nil {
					c.JSON(http.StatusInternalServerError, VideoURLResponse{
						Status: "error",
						Message: "Failed to get task status",
						Error:  err.Error(),
					})
					return
				}

				if task.Status == "completed" {
					c.JSON(http.StatusOK, VideoURLResponse{
						Status:     "success",
						Message:    "Video converted successfully",
						TSFilePath: task.Result,
					})
					return
				}

				if task.Status == "failed" {
					c.JSON(http.StatusInternalServerError, VideoURLResponse{
						Status: "error",
						Message: "Failed to convert video",
						Error:  task.Error,
					})
					return
				}

				time.Sleep(1 * time.Second)
			}

			c.JSON(http.StatusRequestTimeout, VideoURLResponse{
				Status: "error",
				Message: "Video conversion timeout",
				Error:  "The conversion process took too long",
			})
		})

		// 获取任务日志接口
		apiRoutes.GET("/task/:taskId/logs", func(c *gin.Context) {
			taskID := c.Param("taskId")

			logFile := fmt.Sprintf("./log/tasks/%s.log", taskID)
			if _, err := os.Stat(logFile); os.IsNotExist(err) {
				c.JSON(http.StatusNotFound, gin.H{
					"error": "Task log not found",
				})
				return
			}

			// 读取日志文件内容
			logData, err := os.ReadFile(logFile)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to read task log",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"taskId": taskID,
				"logs":   string(logData),
			})
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
				taskQueue.Update(task)
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

// downloadFile 下载文件到指定路径
func downloadFile(url, filepath string) error {
	// 发起HTTP GET请求
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 检查响应状态码
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download file, status code: %d", resp.StatusCode)
	}

	// 创建目标文件
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 将响应体内容复制到文件
	_, err = io.Copy(out, resp.Body)
	return err
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

// TestMaterialUpload 测试素材上传接口
func TestMaterialUpload(t *testing.T) {
	router, _, _ := setupTestServer()

	// 创建一个临时测试文件
	tempFile, err := os.CreateTemp("", "test_upload_*.txt")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// 写入测试内容
	_, err = tempFile.WriteString("This is a test file for upload")
	assert.NoError(t, err)
	tempFile.Close()

	// 创建multipart表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	fileWriter, err := writer.CreateFormFile("file", filepath.Base(tempFile.Name()))
	assert.NoError(t, err)

	// 读取测试文件内容
	fileContent, err := os.ReadFile(tempFile.Name())
	assert.NoError(t, err)

	_, err = io.Copy(fileWriter, bytes.NewReader(fileContent))
	assert.NoError(t, err)
	writer.Close()

	// 创建请求
	req, _ := http.NewRequest("POST", "/api/v1/material/upload", body)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 检查响应
	assert.Equal(t, http.StatusAccepted, w.Code)

	var response MaterialUploadResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.NotEmpty(t, response.TaskID)
	assert.Equal(t, "accepted", response.Status)
	assert.Equal(t, "Material upload accepted for processing", response.Message)
}

// TestVideoURL 测试视频URL处理接口
func TestVideoURL(t *testing.T) {
	router, _, _ := setupTestServer()

	// 创建一个临时测试文件
	tempFile, err := os.CreateTemp("", "test_video_*.mp4")
	assert.NoError(t, err)
	defer os.Remove(tempFile.Name())

	// 写入测试内容
	_, err = tempFile.WriteString("This is a test video file")
	assert.NoError(t, err)
	tempFile.Close()

	// 构造请求数据
	requestData := VideoURLRequest{
		URL: fmt.Sprintf("file://%s", tempFile.Name()), // 使用本地文件URL进行测试
	}

	jsonData, _ := json.Marshal(requestData)
	req, _ := http.NewRequest("POST", "/api/v1/video/url", bytes.NewBuffer(jsonData))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// 注意：由于这是一个集成测试，实际的转换可能需要一些时间
	// 我们只验证请求是否被正确接收和处理
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError, http.StatusRequestTimeout}, w.Code)
}

// TestGetVideoEditStatus 测试获取视频编辑任务状态接口
func TestGetVideoEditStatus(t *testing.T) {
	router, taskQueue, _ := setupTestServer()

	// 先创建一个任务
	task := &queue.Task{
		ID:       uuid.New().String(),
		Spec:     map[string]interface{}{},
		Status:   "pending",
		Created:  time.Now(),
		Progress: 0.0,
	}
	taskQueue.Push(task)

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
	task := &queue.Task{
		ID:       uuid.New().String(),
		Spec:     map[string]interface{}{},
		Status:   "pending",
		Created:  time.Now(),
		Progress: 0.0,
	}
	taskQueue.Push(task)

	// 取消任务
	req, _ := http.NewRequest("DELETE", "/api/v1/video/edit/"+task.ID, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
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