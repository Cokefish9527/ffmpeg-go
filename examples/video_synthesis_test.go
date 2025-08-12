package example

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
)

// TestVideoSynthesis 测试视频合成接口
func TestVideoSynthesis(t *testing.T) {
	// 初始化任务队列
	taskQueue := queue.NewInMemoryTaskQueue()

	// 初始化工作池
	workerPool := service.NewWorkerPool(1, taskQueue)
	workerPool.Start()
	defer workerPool.Stop()

	// 创建Gin引擎
	gin.SetMode(gin.TestMode)
	router := gin.New()

	// 注册视频编辑路由
	router.POST("/api/v1/video/edit", func(c *gin.Context) {
		var req api.VideoEditRequest
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
			Progress: 0.0,
			Verbose:  req.Verbose, // 设置详细日志开关
		}

		// 将任务添加到队列
		if err := taskQueue.Push(task); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to add task to queue",
			})
			return
		}

		// 返回成功响应
		response := api.VideoEditResponse{
			TaskID:  taskID,
			Status:  "accepted",
			Message: "Task accepted for processing",
		}

		c.JSON(http.StatusAccepted, response)
	})

	// 获取任务状态路由
	router.GET("/api/v1/video/edit/:id", func(c *gin.Context) {
		taskID := c.Param("id")
		task, err := taskQueue.Get(taskID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to get task",
			})
			return
		}

		if task == nil {
			c.JSON(http.StatusNotFound, gin.H{
				"error": "Task not found",
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"taskId":   task.ID,
			"status":   task.Status,
			"progress": task.Progress,
			"error":    task.Error,
		})
	})

	// 构造视频合成请求参数
	videoEditRequest := map[string]interface{}{
		"spec": map[string]interface{}{
			"outPath": "./test_output.mp4",
			"width":   1920,
			"height":  1080,
			"fps":     30,
			"verbose": true, // 启用详细日志
			"defaults": map[string]interface{}{
				"duration": 3,
			},
			"clips": []map[string]interface{}{
				{
					"layers": []map[string]interface{}{
						{
							"type": "video",
							"path": "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
						},
					},
				},
				{
					"layers": []map[string]interface{}{
						{
							"type": "video",
							"path": "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
						},
					},
				},
				{
					"layers": []map[string]interface{}{
						{
							"type": "video",
							"path": "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
						},
					},
				},
				{
					"layers": []map[string]interface{}{
						{
							"type": "video",
							"path": "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
						},
					},
				},
				{
					"layers": []map[string]interface{}{
						{
							"type": "video",
							"path": "http://aima-hotvideogeneration-mp4tots.oss-cn-hangzhou.aliyuncs.com/2%2Fa40ea039-c471-4e2b-a9fb-d2065a547391.ts?Expires=1754685363&OSSAccessKeyId=LTAI5tFufCghCDEMueTE88Ba&Signature=k28%2FafMYXiF9InlvFaWyZjqxIj4%3D",
						},
					},
				},
			},
			"keepSourceAudio": true,
		},
		"verbose":    true,
		"outputPath": "./test_output.mp4",
	}

	// 序列化请求体
	requestBody, err := json.Marshal(videoEditRequest)
	assert.NoError(t, err)

	// 发送POST请求提交视频编辑任务
	w := performRequest(router, "POST", "/api/v1/video/edit", bytes.NewReader(requestBody))
	assert.Equal(t, http.StatusAccepted, w.Code)

	// 解析响应
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 获取任务ID
	taskID, ok := response["taskId"].(string)
	assert.True(t, ok)
	assert.NotEmpty(t, taskID)

	fmt.Printf("任务已提交，任务ID: %s\n", taskID)

	// 轮询任务状态
	for i := 0; i < 10; i++ {
		time.Sleep(1 * time.Second)

		// 获取任务状态
		w = performRequest(router, "GET", "/api/v1/video/edit/"+taskID, nil)
		assert.Equal(t, http.StatusOK, w.Code)

		var taskStatus map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &taskStatus)
		assert.NoError(t, err)

		status, ok := taskStatus["status"].(string)
		assert.True(t, ok)

		fmt.Printf("任务状态: %s, 进度: %.2f%%\n", status, taskStatus["progress"].(float64)*100)

		// 如果任务完成或失败，退出循环
		if status == "completed" || status == "failed" {
			break
		}
	}

	// 最终检查任务状态
	task, err := taskQueue.Get(taskID)
	assert.NoError(t, err)
	assert.NotNil(t, task)

	fmt.Printf("最终任务状态: %s\n", task.Status)
	if task.Status == "failed" {
		fmt.Printf("任务失败原因: %s\n", task.Error)
	}
}

// performRequest 辅助函数，用于执行HTTP请求
func performRequest(r http.Handler, method, path string, body *bytes.Reader) *httptest.ResponseRecorder {
	if body == nil {
		body = bytes.NewReader([]byte(""))
	}
	req, _ := http.NewRequest(method, path, body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}