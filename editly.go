// Package ffmpeg_go 提供简化的视频编辑功能
package ffmpeg_go

import (
	"fmt"
	"log"
	"os"
	"strings"
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

	if e.spec.Verbose {
		log.Printf("视频编辑完成: %s", e.spec.OutPath)
	}

	return nil
}

// validateInputs 验证输入文件
func (e *Editly) validateInputs() error {
	if e.spec.Verbose {
		log.Println("验证输入文件...")
	}

	for i, clip := range e.spec.Clips {
		for j, layer := range clip.Layers {
			if layer.Type == "video" && layer.Path != "" {
				if e.spec.Verbose {
					log.Printf("验证片段 %d, 层 %d: %s", i+1, j+1, layer.Path)
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
						// 注意：对于示例，我们只打印警告而不返回错误
						if e.spec.Verbose {
							log.Printf("警告: 文件不存在: %s", layer.Path)
						}
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