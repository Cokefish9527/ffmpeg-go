package api

import (
	"fmt"
	"net/http"
	"os"
	"time"
	
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/u2takey/ffmpeg-go/queue"
	"github.com/u2takey/ffmpeg-go/service"
	"github.com/u2takey/ffmpeg-go/utils"
)

// 用于存储任务队列的全局变量
var globalTaskQueue queue.TaskQueue

// SetTaskQueue 设置全局任务队列
func SetTaskQueue(taskQueue queue.TaskQueue) {
	globalTaskQueue = taskQueue
}

// SubmitVideoEdit 提交视频编辑任务
// @Summary 提交视频编辑任务
// @Description 提交一个新的视频编辑任务
// @Tags video
// @Accept json
// @Produce json
// @Param request body VideoEditRequest true "视频编辑请求"
// @Success 202 {object} VideoEditResponse "任务提交成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /video/edit [post]
func SubmitVideoEdit(c *gin.Context) {
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
		Progress: 0.0,
		Verbose:  req.Verbose, // 设置详细日志开关
		Created:  time.Now(),
	}

	// 将任务添加到队列
	if globalTaskQueue == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Task queue not initialized",
		})
		return
	}

	if err := globalTaskQueue.Push(task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to add task to queue",
		})
		return
	}

	// 创建任务日志记录器
	taskLogger, err := service.NewTaskLogger(taskID)
	if err != nil && req.Verbose {
		fmt.Printf("Failed to create task logger: %v\n", err)
	} else if taskLogger != nil {
		taskLogger.Log("INFO", "任务已提交到队列", map[string]interface{}{
			"taskId":  taskID,
			"status":  "pending",
			"verbose": req.Verbose,
		})
	}

	// 返回成功响应
	response := VideoEditResponse{
		TaskID:  taskID,
		Status:  "accepted",
		Message: "Task accepted for processing",
	}

	c.JSON(http.StatusAccepted, response)
}

// GetVideoEditStatus 获取视频编辑任务状态
// @Summary 获取视频编辑任务状态
// @Description 根据任务ID获取视频编辑任务的状态信息
// @Tags video
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} TaskStatusResponse "任务状态信息"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "任务未找到"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /video/edit/{id} [get]
func GetVideoEditStatus(c *gin.Context) {
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task ID is required",
		})
		return
	}

	if globalTaskQueue == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Task queue not initialized",
		})
		return
	}

	task, err := globalTaskQueue.Get(taskID)
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
		"created":  task.Created,
		"started":  task.Started,
		"finished": task.Finished,
	})
}

// CancelVideoEdit 取消视频编辑任务
// @Summary 取消视频编辑任务
// @Description 根据任务ID取消视频编辑任务
// @Tags video
// @Produce json
// @Param id path string true "任务ID"
// @Success 200 {object} map[string]string "任务取消成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 404 {object} map[string]string "任务未找到"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /video/edit/{id} [delete]
func CancelVideoEdit(c *gin.Context) {
	// 实现取消视频编辑任务的逻辑
	taskID := c.Param("id")
	if taskID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Task ID is required",
		})
		return
	}

	if globalTaskQueue == nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Task queue not initialized",
		})
		return
	}

	task, err := globalTaskQueue.Get(taskID)
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

	// 只能取消待处理和处理中的任务
	if task.Status == "pending" || task.Status == "processing" {
		task.Status = "cancelled"
		task.Finished = time.Now()
		err = globalTaskQueue.Update(task)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Failed to update task",
			})
			return
		}
		
		c.JSON(http.StatusOK, gin.H{
			"message": "Task cancelled successfully",
		})
		return
	}

	c.JSON(http.StatusBadRequest, gin.H{
		"error": "Task cannot be cancelled in current status",
	})
}

// GetWorkerPoolStatus 获取工作池状态
// @Summary 获取工作池状态
// @Description 获取当前工作池的状态信息
// @Tags workerpool
// @Produce json
// @Success 200 {object} map[string]interface{} "工作池状态信息"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /workerpool/status [get]
func GetWorkerPoolStatus(c *gin.Context) {
	// 实现获取工作池状态的逻辑
	c.JSON(http.StatusOK, gin.H{
		"message": "Not implemented",
	})
}

// ResizeWorkerPool 调整工作池大小
// @Summary 调整工作池大小
// @Description 动态调整工作池中工作线程的数量
// @Tags workerpool
// @Accept json
// @Produce json
// @Param request body map[string]int true "工作池大小调整请求"
// @Success 200 {object} map[string]interface{} "工作池调整成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /workerpool/resize [post]
func ResizeWorkerPool(c *gin.Context) {
	// 实现调整工作池大小的逻辑
	c.JSON(http.StatusOK, gin.H{
		"message": "Not implemented",
	})
}

// GetTaskExecutions 获取任务执行历史
// @Summary 获取任务执行历史
// @Description 获取指定任务的执行历史记录
// @Tags monitor
// @Produce json
// @Success 200 {object} map[string]string "任务执行历史"
// @Router /monitor/executions [get]
func GetTaskExecutions(c *gin.Context) {
	// 实现获取任务执行历史的逻辑
	c.JSON(http.StatusOK, gin.H{
		"message": "Task executions",
	})
}

// HandleVideoURL 处理视频URL
// @Summary 处理视频URL
// @Description 通过URL下载视频并提交处理任务
// @Tags video
// @Accept json
// @Produce json
// @Param request body VideoURLRequest true "视频URL请求"
// @Success 200 {object} VideoURLResponse "处理成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /video/url [post]
func HandleVideoURL(c *gin.Context, taskQueue queue.TaskQueue) {
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
	taskID := fmt.Sprintf("t-%s", uuid.New().String())

	// 创建临时目录
	tempDir := "./temp"
	if _, err := os.Stat(tempDir); os.IsNotExist(err) {
		os.Mkdir(tempDir, 0755)
	}

	// 生成临时文件名
	filename := fmt.Sprintf("%s/%s_temp.mp4", tempDir, taskID)

	// 下载文件
	err := utils.DownloadFile(req.URL, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, VideoURLResponse{
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
		c.JSON(http.StatusInternalServerError, VideoURLResponse{
			Status:  "error",
			Message: "Failed to add task to queue",
			Error:   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, VideoURLResponse{
		Status:     "success",
		Message:    "Video converted successfully",
		TSFilePath: outputFile,
		TaskID:     taskID,
	})
}