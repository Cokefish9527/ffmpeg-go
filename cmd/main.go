package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
)

// TaskStatusResponse 任务状态响应
type TaskStatusResponse struct {
	TaskID    string             `json:"taskId"`
	Status    string             `json:"status"`
	Progress  float64            `json:"progress"`
	Message   string             `json:"message,omitempty"`
	Created   string             `json:"created,omitempty"`
	Started   string             `json:"started,omitempty"`
	Finished  string             `json:"finished,omitempty"`
	OutputURL string             `json:"outputUrl,omitempty"`
	Priority  queue.TaskPriority `json:"priority,omitempty"` // 添加优先级字段
}

var (
	taskQueue queue.TaskQueue // 使用queue包中的TaskQueue接口
	editorService service.VideoEditor
	workerPool *service.WorkerPool
)

func main() {
	// 设置Gin运行模式
	gin.SetMode(gin.ReleaseMode)
	
	// 初始化任务队列 (使用queue包中的实现)
	taskQueue = queue.NewInMemoryTaskQueue()
	
	// 从环境变量获取最大工作线程数，默认为0（使用CPU核心数）
	maxWorkers := 0
	if maxWorkersStr := os.Getenv("MAX_WORKERS"); maxWorkersStr != "" {
		if num, err := strconv.Atoi(maxWorkersStr); err == nil {
			maxWorkers = num
		}
	}
	
	// 创建工作池
	workerPool = service.NewWorkerPool(maxWorkers, taskQueue)
	
	// 启动工作池
	workerPool.Start()
	
	// 启动一个goroutine来监听系统信号，用于优雅关闭
	go func() {
		// 创建一个通道来接收系统信号
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		
		// 等待信号
		sig := <-sigChan
		log.Printf("Received signal: %v", sig)
		
		// 停止工作池
		workerPool.Stop()
		
		// 退出程序
		os.Exit(0)
	}()
	
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
		
		// 添加WorkerPool管理接口
		apiGroup.GET("/workerpool/status", getWorkerPoolStatus)
		apiGroup.POST("/workerpool/resize", resizeWorkerPool)
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
	var req service.VideoEditRequest
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
		"priority": task.Priority, // 返回任务优先级
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
	
	response := convertTaskToResponse(task)
	
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

// getWorkerPoolStatus 获取WorkerPool状态
func getWorkerPoolStatus(c *gin.Context) {
	status := gin.H{
		"workerCount": workerPool.GetWorkerCount(),
	}
	
	c.JSON(http.StatusOK, status)
}

// resizeWorkerPool 调整WorkerPool大小
func resizeWorkerPool(c *gin.Context) {
	var req struct {
		Size int `json:"size"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request format",
		})
		return
	}
	
	if req.Size <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Size must be greater than 0",
		})
		return
	}
	
	workerPool.Resize(req.Size)
	
	c.JSON(http.StatusOK, gin.H{
		"message": "WorkerPool resized successfully",
		"workerCount": workerPool.GetWorkerCount(),
	})
}

// convertTaskToResponse 将任务转换为响应格式
func convertTaskToResponse(task *queue.Task) *TaskStatusResponse {
	response := &TaskStatusResponse{
		TaskID:   task.ID,
		Status:   task.Status,
		Progress: task.Progress,
		Message:  task.Error,
		Priority: task.Priority, // 添加优先级字段
	}
	
	if !task.Created.IsZero() {
		response.Created = task.Created.Format(time.RFC3339)
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
	
	return response
}