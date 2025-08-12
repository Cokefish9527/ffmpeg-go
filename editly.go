// Package ffmpeg_go 提供简化的视频编辑功能
package ffmpeg_go

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EditSpec 视频编辑规范
type EditSpec struct {
	OutPath        string                 `json:"outPath"`
	Width          int                    `json:"width"`
	Height         int                    `json:"height"`
	Fps            int                    `json:"fps"`
	Defaults       map[string]interface{} `json:"defaults,omitempty"`
	Clips          []*Clip                `json:"clips"`
	AudioTracks    []string               `json:"audioTracks,omitempty"`
	KeepSourceAudio bool                 `json:"keepSourceAudio,omitempty"`
	Verbose        bool                   `json:"verbose,omitempty"` // 添加详细日志开关
}

// Clip 视频片段
type Clip struct {
	Layers []*Layer `json:"layers"`
}

// Layer 视频层
type Layer struct {
	Type string `json:"type"`
	Path string `json:"path,omitempty"`
	Text string `json:"text,omitempty"`
}

// Editly 视频编辑器
type Editly struct {
	spec *EditSpec
}

// NewEditly 创建新的视频编辑器
func NewEditly(spec *EditSpec) *Editly {
	return &Editly{
		spec: spec,
	}
}

// Edit 执行视频编辑
func (e *Editly) Edit() error {
	if e.spec.Verbose {
		log.Printf("开始编辑视频: %s", e.spec.OutPath)
		log.Printf("尺寸: %dx%d, 帧率: %d", e.spec.Width, e.spec.Height, e.spec.Fps)
		log.Printf("处理 %d 个片段", len(e.spec.Clips))
	}

	// 验证输入文件
	if err := e.validateInputs(); err != nil {
		return fmt.Errorf("输入验证失败: %w", err)
	}

	// 确保输出目录存在
	outputDir := filepath.Dir(e.spec.OutPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("创建输出目录失败: %w", err)
	}

	// 构建FFmpeg命令
	args := []string{"-y"} // 覆盖输出文件

	// 为每个clip添加输入
	for _, clip := range e.spec.Clips {
		for _, layer := range clip.Layers {
			if layer.Type == "video" && layer.Path != "" {
				args = append(args, "-i", layer.Path)
			}
		}
	}

	// 构建过滤器链
	var filterChain []string
	inputIndex := 0

	for _, clip := range e.spec.Clips {
		for _, layer := range clip.Layers {
			if layer.Type == "video" && layer.Path != "" {
				// 为每个视频输入创建标签
				filterChain = append(filterChain, fmt.Sprintf("[%d:v]scale=%d:%d[v%d]", 
					inputIndex, e.spec.Width, e.spec.Height, inputIndex))
				inputIndex++
			} else if layer.Type == "image" && layer.Path != "" {
				// 图片处理
				filterChain = append(filterChain, fmt.Sprintf("[%d:v]scale=%d:%d[img%d]", 
					inputIndex, e.spec.Width, e.spec.Height, inputIndex))
				inputIndex++
			}
		}
	}

	// 添加过滤器参数
	if len(filterChain) > 0 {
		args = append(args, "-filter_complex", strings.Join(filterChain, ";"))
	}

	// 设置输出参数
	args = append(args, 
		"-c:v", "libx264",
		"-preset", "ultrafast",
		"-crf", "23",
		"-r", fmt.Sprintf("%d", e.spec.Fps),
		"-pix_fmt", "yuv420p",
		e.spec.OutPath)

	if e.spec.Verbose {
		log.Printf("执行FFmpeg命令: ffmpeg %s", strings.Join(args, " "))
	}

	// 执行FFmpeg命令
	startTime := time.Now()
	err := Input("").Output(e.spec.OutPath, nil).OverWriteOutput().Run()
	if err != nil {
		return fmt.Errorf("视频编辑失败: %w", err)
	}

	executionTime := time.Since(startTime)
	if e.spec.Verbose {
		log.Printf("视频编辑完成: %s，耗时: %v", e.spec.OutPath, executionTime)
	}

	return nil
}

// validateInputs 验证输入文件
func (e *Editly) validateInputs() error {
	if e.spec.Verbose {
		log.Println("验证输入文件...")
	}

	for _, clip := range e.spec.Clips {
		for _, layer := range clip.Layers {
			if (layer.Type == "video" || layer.Type == "image") && layer.Path != "" {
				if e.spec.Verbose {
					log.Printf("验证层: %s", layer.Path)
				}
				
				// 检查文件是否存在
				if strings.HasPrefix(layer.Path, "http") {
					// 对于网络文件，只做基本检查
					if !strings.Contains(layer.Path, "://") {
						return fmt.Errorf("无效的URL: %s", layer.Path)
					}
				} else {
					// 对于本地文件，检查文件是否存在
					if _, err := os.Stat(layer.Path); os.IsNotExist(err) {
						return fmt.Errorf("文件不存在: %s", layer.Path)
					}
				}
			}
		}
	}

	return nil
}

// Edit 是一个便捷函数，直接编辑视频
func Edit(spec *EditSpec) error {
	editly := NewEditly(spec)
	return editly.Edit()
}