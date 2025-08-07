package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
	"github.com/u2takey/ffmpeg-go/utils"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
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

// VideoURLRequest 视频URL请求结构体
type VideoURLRequest struct {
	URL string `json:"url"`
}

// VideoURLResponse 视频URL响应结构体
type VideoURLResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	TSFilePath string `json:"tsFilePath,omitempty"`
	Error      string `json:"error,omitempty"`
	TaskID     string `json:"taskId,omitempty"`
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

var (
	taskQueue     queue.TaskQueue     // 使用queue包中的TaskQueue接口
	editorService service.VideoEditor // 视频编辑服务
	workerPool    *service.WorkerPool // 工作池
	monitorAPI    *api.MonitorAPI     // 监控API
)

// loadTasksFromFile 从文件加载任务
func loadTasksFromFile(filename string, taskQueue queue.TaskQueue) {
	// 检查文件是否存在
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		utils.Info("任务文件不存在，跳过加载", map[string]string{"filename": filename})
		return
	}

	// 读取文件内容
	data, err := os.ReadFile(filename)
	if err != nil {
		utils.Error("读取任务文件失败", map[string]string{"filename": filename, "error": err.Error()})
		return
	}

	// 解析JSON数据
	var tasks []queue.Task
	if err := json.Unmarshal(data, &tasks); err != nil {
		utils.Error("解析任务文件失败", map[string]string{"filename": filename, "error": err.Error()})
		return
	}

	// 将任务添加到队列
	for _, task := range tasks {
		// 创建任务副本以避免指针问题
		taskCopy := task
		if err := taskQueue.Push(&taskCopy); err != nil {
			utils.Error("添加任务到队列失败", map[string]string{"taskId": task.ID, "error": err.Error()})
		}
	}

	utils.Info("任务加载完成", map[string]string{"filename": filename, "taskCount": fmt.Sprintf("%d", len(tasks))})
}

func main() {
	// 初始化全局日志记录器
	utils.InitGlobalLogger()
	utils.Info("服务启动中", map[string]string{"phase": "initialization"})

	// 设置Gin运行模式
	gin.SetMode(gin.ReleaseMode)

	// 初始化任务队列 (使用持久化任务队列)
	var err error
	taskQueue, err = queue.NewPersistentTaskQueue("./data")
	if err != nil {
		utils.Error("任务队列初始化失败", map[string]string{"error": err.Error()})
		log.Fatal("Failed to initialize task queue:", err)
	}

	// 初始化工作池
	maxWorkers, _ := strconv.Atoi(os.Getenv("MAX_WORKERS"))
	if maxWorkers <= 0 {
		maxWorkers = 12 // 默认12个工作者
	}

	workerPool = service.NewWorkerPool(maxWorkers, taskQueue)
	utils.Info("工作池初始化完成", map[string]string{"maxWorkers": strconv.Itoa(maxWorkers)})

	// 初始化视频编辑服务
	editorService = service.NewVideoEditorService(taskQueue)
	utils.Info("视频编辑服务初始化完成", nil)

	// 初始化监控API
	monitorAPI = api.NewMonitorAPI(taskQueue, workerPool)
	utils.Info("监控API初始化完成", nil)

	// 创建Gin引擎
	r := gin.Default()

	// 静态资源路由，暴露web目录
	r.Static("/web", "./web")
	// Swagger UI路由，暴露swagger目录
	r.Static("/swagger", "./web/swagger/dist")

	// 添加日志中间件
	r.Use(gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		return fmt.Sprintf("[GIN] %s | %3d | %13v | %15s | %-7s %s\n",
			param.TimeStamp.Format("2006/01/02 - 15:04:05"),
			param.StatusCode,
			param.Latency,
			param.ClientIP,
			param.Method,
			param.Path,
		)
	}))
	r.Use(gin.Recovery())

	// 健康检查端点
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	// API路由组
	apiGroup := r.Group("/api/v1")
	{
		// Swagger 文档路由
		apiGroup.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		// 修复路由冲突问题，将具体的doc.json和doc.yaml路径移到不同的前缀下
		r.StaticFile("/api/v1/swagger-doc/doc.json", "./docs/swagger.json")
		r.StaticFile("/api/v1/swagger-doc/doc.yaml", "./docs/swagger.yaml")
		
		// 视频编辑相关接口
		apiGroup.POST("/video/edit", func(c *gin.Context) {
			var req api.VideoEditRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid request format",
				})
				return
			}

			// 提交任务到视频编辑服务
			taskReq := &service.VideoEditRequest{
				Spec:     req.Spec,
				Priority: queue.TaskPriority(req.Priority), // 类型转换
			}

			task, err := editorService.SubmitTask(taskReq)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to submit task",
				})
				return
			}

			c.JSON(http.StatusAccepted, gin.H{
				"taskId":  task.ID,
				"status":  "accepted",
				"message": "Task accepted for processing",
			})
		})

		// 获取任务状态
		apiGroup.GET("/video/edit/:taskId", func(c *gin.Context) {
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

			response := TaskStatusResponse{
				TaskID:   task.ID,
				Status:   task.Status,
				Progress: task.Progress,
				Message:  task.Error,
				Priority: task.Priority,
			}

			if !task.Created.IsZero() {
				response.Created = task.Created.Format(time.RFC3339)
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

		// 取消任务
		apiGroup.DELETE("/video/edit/:taskId", func(c *gin.Context) {
			taskID := c.Param("taskId")

			err := editorService.CancelTask(taskID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Failed to cancel task",
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"message": "Task cancelled successfully",
			})
		})

		// 视频URL处理接口
		apiGroup.POST("/video/url", func(c *gin.Context) {
			var req VideoURLRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, VideoURLResponse{
					Status:  "error",
					Message: "Invalid request format",
					Error:   err.Error(),
				})
				return
			}

			if req.URL == "" {
				c.JSON(http.StatusBadRequest, VideoURLResponse{
					Status:  "error",
					Message: "URL is required",
					Error:   "URL field is empty",
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
			
			// 创建一个带超时的上下文，限制下载时间
			downloadCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			
			// 使用goroutine下载文件，避免阻塞API
			downloadCh := make(chan error, 1)
			go func() {
				downloadCh <- downloadFile(req.URL, filename)
			}()
			
			// 等待下载完成或超时
			var downloadErr error
			select {
			case downloadErr = <-downloadCh:
				// 下载完成
			case <-downloadCtx.Done():
				// 下载超时
				downloadErr = fmt.Errorf("download timeout")
			}
			
			if downloadErr != nil {
				// 清理已创建的文件
				os.Remove(filename)
				c.JSON(http.StatusInternalServerError, VideoURLResponse{
					Status:  "error",
					Message: "Failed to download file",
					Error:   downloadErr.Error(),
				})
				return
			}

			// 生成输出文件路径 (TS格式)
			ext := filepath.Ext(filename)
			outputFile := filename[0:len(filename)-len(ext)] + ".ts"

			// 创建任务对象，与素材预处理器兼容
			task := &queue.Task{
				ID: taskID,
				Spec: map[string]interface{}{
					"source":   filename,
					"taskType": "materialPreprocess",
				},
				Status:   "pending",
				Created:  time.Now(),
				Progress: 0.0,
			}

			// 将任务添加到队列
			if err := taskQueue.Push(task); err != nil {
				// 清理已下载的文件
				os.Remove(filename)
				c.JSON(http.StatusInternalServerError, VideoURLResponse{
					Status:  "error",
					Message: "Failed to add task to queue",
					Error:   err.Error(),
				})
				return
			}

			// 使用goroutine处理转换任务，提高API响应速度
			go func() {
				// 等待任务完成（最多等待60秒）
				timeout := time.After(60 * time.Second)
				ticker := time.NewTicker(1 * time.Second)
				defer ticker.Stop()

				for {
					select {
					case <-timeout:
						// 超时处理
						task.Status = "failed"
						task.Error = "Video conversion timeout"
						taskQueue.Update(task)
						// 清理文件
						os.Remove(filename)
						os.Remove(outputFile)
						return
					case <-ticker.C:
						updatedTask, err := taskQueue.Get(taskID)
						if err != nil {
							// 获取任务状态失败
							task.Status = "failed"
							task.Error = "Failed to get task status"
							taskQueue.Update(task)
							// 清理文件
							os.Remove(filename)
							os.Remove(outputFile)
							return
						}

						if updatedTask.Status == "completed" {
							// 转换成功完成，文件清理将在适当时机进行
							return
						}

						if updatedTask.Status == "failed" {
							// 转换失败，清理文件
							os.Remove(filename)
							os.Remove(outputFile)
							return
						}
						// 任务仍在处理中，继续等待
					}
				}
			}()

			// 立即返回任务已接受的响应
			c.JSON(http.StatusAccepted, VideoURLResponse{
				Status:     "accepted",
				Message:    "Video conversion task accepted",
				TSFilePath: outputFile,
				TaskID:     taskID,
			})
		})
		
		// 查询视频URL转换任务状态接口
		apiGroup.GET("/video/url/:taskId", func(c *gin.Context) {
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
			}
			
			if !task.Created.IsZero() {
				response.Created = task.Created.Format(time.RFC3339)
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
	}

	// 监控相关接口
	monitorGroup := r.Group("/api/v1/monitor")
	{
		// 系统统计信息
		monitorGroup.GET("/stats", monitorAPI.GetSystemStats)

		// 任务统计信息
		monitorGroup.GET("/tasks/stats", monitorAPI.GetTaskStats)

		// 任务列表
		monitorGroup.GET("/tasks", monitorAPI.GetTasks)

		// 任务详情
		monitorGroup.GET("/tasks/:taskId", monitorAPI.GetTaskDetail)

		// Worker统计信息
		monitorGroup.GET("/workers", monitorAPI.GetWorkerStats)

		// 重试任务
		monitorGroup.POST("/tasks/retry", monitorAPI.RetryTask)

		// 取消任务
		monitorGroup.POST("/tasks/cancel", monitorAPI.CancelTask)

		// 丢弃任务
		monitorGroup.POST("/tasks/discard", monitorAPI.DiscardTask)

		// 任务执行历史
		monitorGroup.GET("/tasks/:taskId/executions", monitorAPI.GetTaskExecutions)
	}

	// 启动工作池
	workerPool.Start()
	utils.Info("工作池已启动", map[string]string{"workerCount": fmt.Sprintf("%d", workerPool.GetWorkerCount())})

	// 从文件加载任务
	loadTasksFromFile("./data/tasks.json", taskQueue)

	// 启动HTTP服务器
	port := os.Getenv("PORT")
	if port == "" {
		port = "8082" // 默认端口改为8082
	}

	utils.Info("服务器启动中", map[string]string{"port": port})

	srv := &http.Server{
		Addr:    ":" + port,
		Handler: r,
	}

	// 在goroutine中启动服务器
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			utils.Error("服务器启动失败", map[string]string{"error": err.Error()})
			log.Fatalf("listen: %s\n", err)
		}
	}()

	utils.Info("服务器已启动", map[string]string{"port": port})

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	utils.Info("正在关闭服务器...", nil)

	// 关闭工作池
	workerPool.Stop()
	utils.Info("工作池已关闭", nil)

	// 关闭服务器
	if err := srv.Shutdown(nil); err != nil {
		utils.Error("服务器关闭失败", map[string]string{"error": err.Error()})
		log.Fatal("Server forced to shutdown:", err)
	}

	utils.Info("服务器已退出", nil)
}
