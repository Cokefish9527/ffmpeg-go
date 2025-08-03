package main

import (
	"log"
	"net/http"
	"os"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/service"
)

var (
	taskQueue service.TaskQueue
	editorService service.VideoEditor
	workerPool *service.WorkerPool
)

func main() {
	// 设置Gin运行模式
	gin.SetMode(gin.ReleaseMode)
	
	// 初始化任务队列
	taskQueue = service.NewInMemoryTaskQueue()
	
	// 创建工作池
	workerPool = service.NewWorkerPool(0, taskQueue) // 0表示使用CPU核心数
	
	// 启动工作池
	workerPool.Start()
	
	// 初始化视频编辑服务
	editorService = service.NewVideoEditorService(taskQueue)
	
	// 创建Gin引擎
	r := gin.Default()
	
	// 定义健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})
	
	// 定义API路由组
	apiGroup := r.Group("/api/v1")
	{
		apiGroup.POST("/video/edit", submitVideoEdit)
		apiGroup.GET("/video/edit/:taskId", getVideoEditStatus)
		apiGroup.DELETE("/video/edit/:taskId", cancelVideoEdit)
	}
	
	// 从环境变量获取端口，默认为8082
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082"
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
	
	// 创建任务
	task, err := editorService.SubmitTask(&req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to submit task",
		})
		return
	}
	
	c.JSON(http.StatusAccepted, gin.H{
		"taskId": task.ID,
		"status": task.Status,
	})
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
	}
	
	if task.Result != "" {
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
			"error": "Failed to cancel task",
			"taskId": taskID,
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"message": "Task cancelled successfully",
		"taskId": taskID,
	})
}