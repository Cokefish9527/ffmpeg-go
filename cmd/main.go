package main

import (
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"

	// 导入swagger文档
	_ "github.com/u2takey/ffmpeg-go/docs"
)

// downloadFile 下载文件到指定路径
func downloadFile(url, filepath string) error {
	// 发起HTTP GET请求
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// 创建目标文件
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// 将HTTP响应内容写入文件
	_, err = io.Copy(out, resp.Body)
	return err
}

func main() {
	// 初始化任务队列
	taskQueue := queue.NewInMemoryTaskQueue()
	
	// 初始化工作池
	workerPool := service.NewWorkerPool(3, taskQueue)
	
	// 初始化监控API
	monitorAPI := api.NewMonitorAPI(taskQueue, workerPool)

	// 启动工作池
	workerPool.Start()

	// 确保程序退出时停止工作池
	defer workerPool.Stop()

	// 启动HTTP服务器
	router := gin.Default()

	// 提供静态文件服务
	router.StaticFile("/", "./web/index.html")
	router.Static("/static", "./web")
	
	// 添加Swagger路由
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := router.Group("/api/v1")
	{
		v1.POST("/video/edit", api.SubmitVideoEdit)
		v1.GET("/video/edit/:id", api.GetVideoEditStatus)
		v1.DELETE("/video/edit/:id", api.CancelVideoEdit)
		v1.GET("/workerpool/status", api.GetWorkerPoolStatus)
		v1.POST("/workerpool/resize", api.ResizeWorkerPool)

		// 添加监控接口
		v1.GET("/monitor/stats", monitorAPI.GetSystemStats)
		v1.GET("/monitor/tasks/stats", monitorAPI.GetTaskStats)
		v1.GET("/monitor/tasks", monitorAPI.GetTasks)
		v1.GET("/monitor/tasks/:taskId", monitorAPI.GetTaskDetail)
		v1.GET("/monitor/tasks/:taskId/executions", monitorAPI.GetTaskExecutions)
		v1.GET("/monitor/tasks/:taskId/log", monitorAPI.GetTaskLog) // 添加获取任务日志的接口
		v1.GET("/monitor/workers", monitorAPI.GetWorkerStats)

		// 添加任务管理接口
		v1.POST("/monitor/tasks/retry", monitorAPI.RetryTask)
		v1.POST("/monitor/tasks/cancel", monitorAPI.CancelTask)
		v1.POST("/monitor/tasks/discard", monitorAPI.DiscardTask)

		// 视频URL处理接口
		v1.POST("/video/url", func(c *gin.Context) {
			var req api.VideoURLRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(400, api.VideoURLResponse{
					Status:  "error",
					Message: "Invalid request format",
					Error:   err.Error(),
				})
				return
			}

			if req.URL == "" {
				c.JSON(400, api.VideoURLResponse{
					Status:  "error",
					Message: "URL is required",
					Error:   "URL field is empty",
				})
				return
			}

			// 生成任务ID
			taskID := fmt.Sprintf("t-%s", uuid.New().String())

			// 创建临时目录
			tempDir := "./temp"
			if _, err := os.Stat(tempDir); os.IsNotExist(err) {
				os.Mkdir(tempDir, 0755)
			}

			// 生成临时文件名
			filename := fmt.Sprintf("%s/%s_temp.mp4", tempDir, taskID)

			// 下载文件
			err := downloadFile(req.URL, filename)
			if err != nil {
				c.JSON(500, api.VideoURLResponse{
					Status:  "error",
					Message: "Failed to download file",
					Error:   err.Error(),
				})
				return
			}

			// 生成输出文件路径 (TS格式)
			ext := ".mp4"
			outputFile := filename[0:len(filename)-len(ext)] + ".ts"

			// 创建任务对象，与素材预处理器兼容
			task := &queue.Task{
				ID: taskID,
				Spec: map[string]interface{}{
					"source":   filename,
					"taskType": "materialPreprocess",
				},
				Status:   "pending",
				Progress: 0.0,
			}

			// 将任务添加到队列
			if err := taskQueue.Push(task); err != nil {
				// 清理已下载的文件
				os.Remove(filename)
				c.JSON(500, api.VideoURLResponse{
					Status:  "error",
					Message: "Failed to add task to queue",
					Error:   err.Error(),
				})
				return
			}

			// 简单示例：处理视频URL
			// 在实际应用中，这里会启动HTTP服务器来处理API请求
			fmt.Println("Video processing service started")

			c.JSON(200, api.VideoURLResponse{
				Status:     "success",
				Message:    "Video converted successfully",
				TSFilePath: outputFile,
				TaskID:     taskID,
			})
		})
	}

	// 启动HTTP服务器监听8084端口
	if err := router.Run(":8084"); err != nil {
		fmt.Printf("Failed to start HTTP server: %v\n", err)
	}
}