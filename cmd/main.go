package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/u2takey/ffmpeg-go/api"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
	swaggerFiles "github.com/swaggo/files"
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

// OSSConfig OSS配置结构体
type OSSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"access_key_id"`
	AccessKeySecret string `json:"access_key_secret"`
	BucketName      string `json:"bucket_name"`
}

// loadOSSConfig 从配置文件加载OSS配置
func loadOSSConfig() *service.OSSConfig {
	config := &service.OSSConfig{}
	
	// 检查配置文件是否存在
	if _, err := os.Stat("config/oss_config.json"); os.IsNotExist(err) {
		fmt.Println("OSS配置文件不存在，使用空配置")
		return config
	}
	
	// 读取配置文件
	data, err := os.ReadFile("config/oss_config.json")
	if err != nil {
		fmt.Println("读取OSS配置文件失败:", err)
		return config
	}
	
	// 解析配置文件
	var ossConfig OSSConfig
	err = json.Unmarshal(data, &ossConfig)
	if err != nil {
		fmt.Println("解析OSS配置文件失败:", err)
		return config
	}
	
	// 转换为服务层配置结构体
	config = &service.OSSConfig{
		Endpoint:        ossConfig.Endpoint,
		AccessKeyID:     ossConfig.AccessKeyID,
		AccessKeySecret: ossConfig.AccessKeySecret,
		BucketName:      ossConfig.BucketName,
	}
	
	return config
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

	// 初始化OSS管理器
	ossConfig := loadOSSConfig()
	ossManager := service.NewOSSManager(*ossConfig)
	ossController := api.NewOSSController(ossManager)

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

		// OSS相关路由
		v1.POST("/oss/upload", ossController.UploadFile)
		v1.GET("/oss/objects", ossController.ListObjects)
		v1.DELETE("/oss/object", ossController.DeleteObject)

		// 视频URL处理接口
		v1.POST("/video/url", func(c *gin.Context) {
			api.HandleVideoURL(c, taskQueue)
		})
	}

	// 启动HTTP服务器监听8084端口
	if err := router.Run(":8084"); err != nil {
		fmt.Printf("Failed to start HTTP server: %v\n", err)
	}
}
