package service

import (
	"fmt"
	"os"
	"path/filepath"

	ffmpeg_go "github.com/u2takey/ffmpeg-go"
	"github.com/u2takey/ffmpeg-go/queue"
)

// MaterialPreprocessor 素材预处理器接口
type MaterialPreprocessor interface {
	Process(task *queue.Task) error
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
	// 更新任务状态为处理中
	task.Status = "processing"
	task.Progress = 0.0

	// 从任务规范中获取源文件路径
	spec, ok := task.Spec.(map[string]interface{})
	if !ok {
		return fmt.Errorf("invalid task spec format")
	}

	source, ok := spec["source"].(string)
	if !ok {
		return fmt.Errorf("missing source file in task spec")
	}

	// 检查源文件是否存在
	if _, err := os.Stat(source); os.IsNotExist(err) {
		return fmt.Errorf("source file does not exist: %s", source)
	}

	// 生成输出文件路径 (TS格式)
	ext := filepath.Ext(source)
	outputFile := source[0:len(source)-len(ext)] + ".ts"

	// 使用FFmpeg将文件转换为TS格式
	err := s.convertToTS(source, outputFile)
	if err != nil {
		task.Status = "failed"
		task.Error = err.Error()
		return err
	}

	// 更新任务状态为已完成
	task.Status = "completed"
	task.Progress = 1.0
	task.Result = outputFile

	return nil
}

// convertToTS 使用FFmpeg将视频文件转换为TS格式
func (s *MaterialPreprocessorService) convertToTS(inputFile, outputFile string) error {
	return ffmpeg_go.Input(inputFile).
		Output(outputFile, ffmpeg_go.KwArgs{
			"c":        "copy",        // 直接复制编解码器
			"bsf:v":    "h264_mp4toannexb", // 视频比特流过滤器
			"f":        "mpegts",      // 输出格式为MPEG-TS
		}).
		OverWriteOutput().
		Run()
}