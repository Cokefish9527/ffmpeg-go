package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
)

var (
	taskQueue     queue.TaskQueue
	editorService service.VideoEditor
	workerPool    *service.WorkerPool
)

func main() {
	// 设置Gin运行模式
	gin.SetMode(gin.ReleaseMode)
	
	// 初始化任务队列
	taskQueue = queue.NewInMemoryTaskQueue()
	
	// 初始化视频编辑服务
	editorService = service.NewVideoEditorService(taskQueue)
	
	// 初始化工作池，最大并发数设为5
	workerPool = service.NewWorkerPool(5)
	
	// 启动工作池
	workerPool.Start()
	
	// 创建Gin引擎
	r := gin.Default()
	
	// 定义健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	
	// 定义API路由组
	apiRoutes := r.Group("/api/v1")
	{
		apiRoutes.POST("/video/edit", submitVideoEdit)
		apiRoutes.GET("/video/edit/:taskId", getVideoEditStatus)
		apiRoutes.DELETE("/video/edit/:taskId", cancelVideoEdit)
	}
	
	// 从环境变量获取端口，默认为8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server:", err)
	}
}

// submitVideoEdit 处理视频编辑任务提交请求
func submitVideoEdit(c *gin.Context) {
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
	response := api.VideoEditResponse{
		TaskID:  taskID,
		Status:  "accepted",
		Message: "Task accepted for processing",
	}
	
	c.JSON(http.StatusAccepted, response)
}

// getVideoEditStatus 获取视频编辑任务状态
func getVideoEditStatus(c *gin.Context) {
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
}

// cancelVideoEdit 取消视频编辑任务
func cancelVideoEdit(c *gin.Context) {
	taskID := c.Param("taskId")
	
	err := editorService.CancelTask(taskID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Failed to cancel task",
			"taskId":  taskID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Task cancelled successfully",
		"taskId":  taskID,
	})
}