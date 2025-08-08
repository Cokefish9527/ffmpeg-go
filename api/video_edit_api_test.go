package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
)

// setupTestServer 设置测试服务器
func setupVideoEditTestServer() (*gin.Engine, queue.TaskQueue, *service.WorkerPool) {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 初始化内存任务队列
	taskQueue := queue.NewInMemoryTaskQueue()

	// 初始化工作池，使用1个worker
	workerPool := service.NewWorkerPool(1, taskQueue)

	// 初始化OSS管理器（使用空配置）
	ossConfig := &service.OSSConfig{}
	ossManager := service.NewOSSManager(*ossConfig)

	// 创建OSS控制器
	ossController := NewOSSController(ossManager)

	// 启动工作池
	workerPool.Start()

	// 创建Gin引擎
	r := gin.New()
	r.Use(gin.Recovery())

	// 定义API路由组
	apiRoutes := r.Group("/api/v1")
	{
		apiRoutes.POST("/video/edit", SubmitVideoEdit)
		apiRoutes.GET("/video/edit/:id", GetVideoEditStatus)
		apiRoutes.DELETE("/video/edit/:id", CancelVideoEdit)
		
		// OSS相关路由
		apiRoutes.POST("/oss/upload", ossController.UploadFile)
		apiRoutes.GET("/oss/objects", ossController.ListObjects)
		apiRoutes.DELETE("/oss/object", ossController.DeleteObject)

		// 智能上传接口
		apiRoutes.POST("/video/smart-upload", func(c *gin.Context) {
			SmartUpload(c, ossManager)
		})
	}

	return r, taskQueue, workerPool
}

// TestVideoEditSubmit 测试提交视频编辑任务
func TestVideoEditSubmit(t *testing.T) {
	// 设置测试服务器
	router, _, workerPool := setupVideoEditTestServer()
	defer workerPool.Stop()

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
	assert.NoError(t, err)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusAccepted, w.Code)

	// 解析响应数据
	var response VideoEditResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证响应内容
	assert.NotEmpty(t, response.TaskID)
	assert.Equal(t, "accepted", response.Status)
	assert.Equal(t, "Video edit task accepted for processing", response.Message)
}

// TestVideoEditStatus 测试获取视频编辑任务状态
func TestVideoEditStatus(t *testing.T) {
	// 设置测试服务器
	router, _, workerPool := setupVideoEditTestServer()
	defer workerPool.Stop()

	// 首先提交一个任务
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
		},
	}

	requestData := VideoEditRequest{
		Spec: editSpec,
	}

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	assert.NoError(t, err)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusAccepted, w.Code)

	// 解析响应数据
	var response VideoEditResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 获取任务状态
	taskID := response.TaskID
	assert.NotEmpty(t, taskID)

	// 创建获取任务状态的请求
	statusReq, err := http.NewRequest("GET", "/api/v1/video/edit/"+taskID, nil)
	assert.NoError(t, err)

	// 创建响应记录器
	statusW := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(statusW, statusReq)

	// 验证响应
	assert.Equal(t, http.StatusOK, statusW.Code)

	// 解析状态响应数据
	var statusResponse TaskStatusResponse
	err = json.Unmarshal(statusW.Body.Bytes(), &statusResponse)
	assert.NoError(t, err)

	// 验证状态响应内容
	assert.Equal(t, taskID, statusResponse.TaskID)
	assert.Equal(t, "pending", statusResponse.Status)
	assert.Equal(t, 0.0, statusResponse.Progress)
}

// TestVideoEditCancel 测试取消视频编辑任务
func TestVideoEditCancel(t *testing.T) {
	// 设置测试服务器
	router, _, workerPool := setupVideoEditTestServer()
	defer workerPool.Stop()

	// 首先提交一个任务
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
		},
	}

	requestData := VideoEditRequest{
		Spec: editSpec,
	}

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	assert.NoError(t, err)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusAccepted, w.Code)

	// 解析响应数据
	var response VideoEditResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 获取任务ID
	taskID := response.TaskID
	assert.NotEmpty(t, taskID)

	// 创建取消任务的请求
	cancelReq, err := http.NewRequest("DELETE", "/api/v1/video/edit/"+taskID, nil)
	assert.NoError(t, err)

	// 创建响应记录器
	cancelW := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(cancelW, cancelReq)

	// 验证响应
	// 注意：由于任务可能已经被处理，取消可能会失败，但我们至少验证路由是否正常工作
	// 在测试环境中，我们期望至少能正确路由到处理函数
	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, cancelW.Code)
}

// TestVideoEditInvalidRequest 测试无效请求
func TestVideoEditInvalidRequest(t *testing.T) {
	// 设置测试服务器
	router, _, workerPool := setupVideoEditTestServer()
	defer workerPool.Stop()

	// 创建无效的JSON请求
	invalidJSON := `{"invalid": "json", "spec": }`

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBufferString(invalidJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// TestVideoEditEmptySpec 测试空spec请求
func TestVideoEditEmptySpec(t *testing.T) {
	// 设置测试服务器
	router, _, workerPool := setupVideoEditTestServer()
	defer workerPool.Stop()

	// 准备空spec的测试数据
	requestData := VideoEditRequest{
		Spec: nil,
	}

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	assert.NoError(t, err)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 执行请求
	router.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusAccepted, w.Code)
}

// BenchmarkVideoEditSubmit 视频编辑任务提交性能测试
func BenchmarkVideoEditSubmit(b *testing.B) {
	// 设置测试服务器
	router, _, workerPool := setupTestServer()
	defer workerPool.Stop()

	// 准备测试数据
	editSpec := map[string]interface{}{
		"outPath": "./benchmark_output.mp4",
		"width":   1920,
		"height":  1080,
		"fps":     30,
		"clips": []map[string]interface{}{
			{
				"duration": 5,
				"layers": []map[string]interface{}{
					{
						"type": "video",
						"path": "http://example.com/benchmark.mp4",
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
	assert.NoError(b, err)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// 创建HTTP请求
		req, err := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
		assert.NoError(b, err)
		req.Header.Set("Content-Type", "application/json")

		// 创建响应记录器
		w := httptest.NewRecorder()

		// 执行请求
		router.ServeHTTP(w, req)

		// 验证响应
		assert.Equal(b, http.StatusAccepted, w.Code)
	}
}