// Package service provides material preprocessing functionality for video files.
// It handles video property detection, format conversion, and task logging.
package service

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
	"github.com/u2takey/ffmpeg-go/queue"
)

// TaskLogger 任务日志记录器
type TaskLogger struct {
	taskID string
	logDir string
}

// VideoProperties 视频属性信息
type VideoProperties struct {
	FileName      string  `json:"fileName"`
	Duration      float64 `json:"duration"`
	Width         int     `json:"width"`
	Height        int     `json:"height"`
	Codec         string  `json:"codec"`
	Bitrate       string  `json:"bitrate"`
	Size          int64   `json:"size"`
	Format        string  `json:"format"`
}

// NewTaskLogger 创建任务日志记录器
func NewTaskLogger(taskID string) (*TaskLogger, error) {
	logDir := "./log/tasks"
	err := os.MkdirAll(logDir, 0755)
	if err != nil {
		return nil, fmt.Errorf("failed to create log directory: %v", err)
	}

	return &TaskLogger{
		taskID: taskID,
		logDir: logDir,
	}, nil
}

// Log 记录日志条目
func (l *TaskLogger) Log(level, message string, data map[string]interface{}) {
	if l == nil {
		return // 安全处理 nil 接收者
	}

	logFile := filepath.Join(l.logDir, fmt.Sprintf("%s.log", l.taskID))

	// 构建日志条目
	logEntry := fmt.Sprintf(
		"%s [%s] %s | %v\n",
		time.Now().Format("2006-01-02 15:04:05"),
		strings.ToUpper(level),
		message,
		data,
	)

	// 追加写入日志文件
	f, err := os.OpenFile(logFile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error opening log file: %v\n", err)
		return
	}
	defer f.Close()

	if _, err := f.WriteString(logEntry); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing to log file: %v\n", err)
	}
}

// GetVideoProperties 获取视频文件属性
func GetVideoProperties(filePath string) (*VideoProperties, error) {
	// 获取文件大小
	var size int64
	if info, err := os.Stat(filePath); err == nil {
		size = info.Size()
	}

	// 使用ffprobe获取视频信息
	probeData, err := ffmpeg_go.Probe(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法探测视频文件属性: %v", err)
	}

	// 解析JSON数据
	var probeResult map[string]interface{}
	if err := json.Unmarshal([]byte(probeData), &probeResult); err != nil {
		return nil, fmt.Errorf("解析视频属性失败: %v", err)
	}

	props := &VideoProperties{
		FileName: filepath.Base(filePath),
		Size:     size,
	}

	// 提取格式信息
	if format, ok := probeResult["format"].(map[string]interface{}); ok {
		if duration, ok := format["duration"].(string); ok {
			fmt.Sscanf(duration, "%f", &props.Duration)
		}
		if formatName, ok := format["format_name"].(string); ok {
			props.Format = formatName
		}
		if bitrate, ok := format["bit_rate"].(string); ok {
			props.Bitrate = bitrate
		}
	}

	// 提取视频流信息
	if streams, ok := probeResult["streams"].([]interface{}); ok {
		for _, stream := range streams {
			if streamMap, ok := stream.(map[string]interface{}); ok {
				if codecType, ok := streamMap["codec_type"].(string); ok && codecType == "video" {
					if width, ok := streamMap["width"].(float64); ok {
						props.Width = int(width)
					}
					if height, ok := streamMap["height"].(float64); ok {
						props.Height = int(height)
					}
					if codec, ok := streamMap["codec_name"].(string); ok {
						props.Codec = codec
					}
					break
				}
			}
		}
	}

	return props, nil
}

// LogFormatConversionTask 记录格式转换任务日志
func (l *TaskLogger) LogFormatConversionTask(task *queue.Task, downloadTime, convertTime float64,
	inputProps, outputProps *VideoProperties) {
	// 记录各阶段耗时
	l.Log("INFO", "格式转换任务各阶段耗时", map[string]interface{}{
		"downloadTime": downloadTime,
		"convertTime":  convertTime,
		"totalTime":    downloadTime + convertTime,
	})

	// 记录转换前后文件属性
	if inputProps != nil {
		l.Log("INFO", "转换前文件属性", map[string]interface{}{
			"fileName": inputProps.FileName,
			"duration": inputProps.Duration,
			"width":    inputProps.Width,
			"height":   inputProps.Height,
			"codec":    inputProps.Codec,
			"bitrate":  inputProps.Bitrate,
			"size":     inputProps.Size,
			"format":   inputProps.Format,
		})
	}

	if outputProps != nil {
		l.Log("INFO", "转换后文件属性", map[string]interface{}{
			"fileName": outputProps.FileName,
			"duration": outputProps.Duration,
			"width":    outputProps.Width,
			"height":   outputProps.Height,
			"codec":    outputProps.Codec,
			"bitrate":  outputProps.Bitrate,
			"size":     outputProps.Size,
			"format":   outputProps.Format,
		})
	}
}

// MaterialPreprocessor 素材预处理器接口
type MaterialPreprocessor interface {
	Process(task *queue.Task) error
}

// CallbackRequest 回调请求结构
type CallbackRequest struct {
	TaskID   string `json:"taskId"`
	Status   string `json:"status"`
	Result   string `json:"result,omitempty"`
	Error    string `json:"error,omitempty"`
}

// MaterialPreprocessorService 素材预处理器服务
type MaterialPreprocessorService struct {
}

// NewMaterialPreprocessorService 创建素材预处理器服务实例
func NewMaterialPreprocessorService() MaterialPreprocessor {
	return &MaterialPreprocessorService{}
}

// Process 处理素材预处理任务
func (s *MaterialPreprocessorService) Process(task *queue.Task) error {
	// 创建任务日志记录器
	taskLogger, err := NewTaskLogger(task.ID)
	if err != nil {
		// 即使日志记录器创建失败，也继续处理任务
	}

	if taskLogger != nil {
		taskLogger.Log("INFO", "开始处理素材预处理任务", map[string]interface{}{
			"taskId": task.ID,
			"status": "processing",
		})
	}

	// 记录开始时间
	startTime := time.Now()

	// 更新任务状态为处理中
	task.Status = "processing"
	task.Progress = 0.0

	// 从任务规范中获取源文件路径
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		err := fmt.Errorf("invalid task spec format")
		if taskLogger != nil {
			taskLogger.Log("ERROR", "任务规范格式无效", map[string]interface{}{"error": err.Error()})
		}
		s.sendCallback(task, "failed", "", err.Error())
		return err
	}

	source, ok := spec["source"].(string)
	if !ok {
		err := fmt.Errorf("missing source file in task spec")
		if taskLogger != nil {
			taskLogger.Log("ERROR", "任务规范中缺少源文件", map[string]interface{}{"error": err.Error()})
		}
		s.sendCallback(task, "failed", "", err.Error())
		return err
	}

	// 检查源文件是否存在
	fileCheckStart := time.Now()
	if _, err := os.Stat(source); os.IsNotExist(err) {
		fileCheckDuration := time.Since(fileCheckStart).Seconds()
		if taskLogger != nil {
			taskLogger.Log("ERROR", "源文件不存在", map[string]interface{}{
				"source": source,
				"error":  err.Error(),
				"duration": fileCheckDuration,
			})
		}
		s.sendCallback(task, "failed", "", err.Error())
		return fmt.Errorf("source file does not exist: %s", source)
	}
	fileCheckDuration := time.Since(fileCheckStart).Seconds()

	if taskLogger != nil {
		taskLogger.Log("INFO", "源文件检查完成", map[string]interface{}{
			"source": source,
			"duration": fileCheckDuration,
		})
	}

	// 获取转换前文件属性
	getPropsStart := time.Now()
	var inputProps *VideoProperties
	if taskLogger != nil {
		props, err := GetVideoProperties(source)
		if err == nil {
			inputProps = props
			taskLogger.Log("INFO", "转换前文件属性", map[string]interface{}{
				"fileName": props.FileName,
				"duration": props.Duration,
				"width":    props.Width,
				"height":   props.Height,
				"codec":    props.Codec,
				"bitrate":  props.Bitrate,
				"size":     props.Size,
				"format":   props.Format,
			})
		} else {
			taskLogger.Log("WARN", "无法获取转换前文件属性", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	getPropsDuration := time.Since(getPropsStart).Seconds()
	if taskLogger != nil {
		taskLogger.Log("INFO", "获取源文件属性耗时", map[string]interface{}{
			"duration": getPropsDuration,
		})
	}

	// 生成输出文件路径 (TS格式)
	pathGenStart := time.Now()
	ext := filepath.Ext(source)
	outputFile := source[0:len(source)-len(ext)] + ".ts"
	pathGenDuration := time.Since(pathGenStart).Seconds()

	if taskLogger != nil {
		taskLogger.Log("INFO", "输出路径生成完成", map[string]interface{}{
			"outputFile": outputFile,
			"duration": pathGenDuration,
		})
	}

	// 记录转换开始时间
	conversionStart := time.Now()

	// 使用FFmpeg将文件转换为TS格式
	err = s.convertToTS(source, outputFile, taskLogger)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		if taskLogger != nil {
			taskLogger.Log("ERROR", "格式转换失败", map[string]interface{}{
				"error": err.Error(),
			})
		}
		s.sendCallback(task, "failed", "", err.Error())
		return err
	}

	// 记录转换结束时间
	conversionEnd := time.Now()
	conversionDuration := conversionEnd.Sub(conversionStart).Seconds()

	if taskLogger != nil {
		taskLogger.Log("INFO", "格式转换完成", map[string]interface{}{
			"duration": conversionDuration,
		})
	}

	// 获取转换后文件属性
	getOutputPropsStart := time.Now()
	var outputProps *VideoProperties
	if taskLogger != nil {
		props, err := GetVideoProperties(outputFile)
		if err == nil {
			outputProps = props
			taskLogger.Log("INFO", "转换后文件属性", map[string]interface{}{
				"fileName": props.FileName,
				"duration": props.Duration,
				"width":    props.Width,
				"height":   props.Height,
				"codec":    props.Codec,
				"bitrate":  props.Bitrate,
				"size":     props.Size,
				"format":   props.Format,
			})
		} else {
			taskLogger.Log("WARN", "无法获取转换后文件属性", map[string]interface{}{
				"error": err.Error(),
			})
		}
	}
	getOutputPropsDuration := time.Since(getOutputPropsStart).Seconds()
	if taskLogger != nil {
		taskLogger.Log("INFO", "获取输出文件属性耗时", map[string]interface{}{
			"duration": getOutputPropsDuration,
		})
	}

	// 记录任务处理总时间
	endTime := time.Now()
	totalDuration := endTime.Sub(startTime).Seconds()

	if taskLogger != nil {
		taskLogger.Log("INFO", "任务处理完成", map[string]interface{}{
			"totalDuration": totalDuration,
			"fileCheckDuration": fileCheckDuration,
			"getPropsDuration": getPropsDuration,
			"pathGenDuration": pathGenDuration,
			"conversionDuration": conversionDuration,
			"getOutputPropsDuration": getOutputPropsDuration,
		})

		// 记录格式转换任务详细日志
		taskLogger.LogFormatConversionTask(task, 0, conversionDuration, inputProps, outputProps)
	}

	// 更新任务状态为已完成
	task.Status = "completed"
	task.Progress = 1.0
	task.Result = outputFile

	s.sendCallback(task, "completed", outputFile, "")

	return nil
}

// convertToTS 使用FFmpeg将视频文件转换为TS格式
func (s *MaterialPreprocessorService) convertToTS(inputFile, outputFile string, taskLogger *TaskLogger) error {
	// 记录FFmpeg命令构建时间
	buildCmdStart := time.Now()

	ffmpeg := ffmpeg_go.Input(inputFile).
		Output(outputFile, ffmpeg_go.KwArgs{
			"c":        "copy",        // 直接复制编解码器
			"bsf:v":    "h264_mp4toannexb", // 视频比特流过滤器
			"f":        "mpegts",      // 输出格式为MPEG-TS
		}).
		OverWriteOutput()

	buildCmdDuration := time.Since(buildCmdStart).Seconds()
	if taskLogger != nil {
		taskLogger.Log("INFO", "FFmpeg命令构建完成", map[string]interface{}{
			"duration": buildCmdDuration,
		})
	}

	// 记录FFmpeg执行时间
	execStart := time.Now()
	err := ffmpeg.Run()
	execDuration := time.Since(execStart).Seconds()

	if taskLogger != nil {
		taskLogger.Log("INFO", "FFmpeg执行完成", map[string]interface{}{
			"duration": execDuration,
			"success": err == nil,
		})
	}

	return err
}

// sendCallback 发送回调通知
func (s *MaterialPreprocessorService) sendCallback(task *queue.Task, status, result, errorMsg string) {
	// 从任务规范中获取回调URL
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		return
	}

	callbackURL, ok := spec["callback"].(string)
	if !ok || callbackURL == "" {
		// 没有回调URL，直接返回
		return
	}

	// 构造回调请求
	callbackReq := CallbackRequest{
		TaskID: task.ID,
		Status: status,
		Result: result,
		Error:  errorMsg,
	}

	// 序列化请求体
	jsonData, err := json.Marshal(callbackReq)
	if err != nil {
		return
	}

	// 发送POST请求
	resp, err := http.Post(callbackURL, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return
	}
	defer resp.Body.Close()
	
	// 可以根据需要处理响应，这里简单地忽略
	_ = resp
}