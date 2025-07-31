// Package ffmpeg_go 提供对标editly的高级视频编辑功能
package ffmpeg_go

import (
	"encoding/json"
	"fmt"
	"image/color"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// EditSpec 定义视频编辑规范，类似于editly的spec
type EditSpec struct {
	OutPath          string        `json:"outPath"`
	Width            int           `json:"width"`
	Height           int           `json:"height"`
	Fps              int           `json:"fps"`
	Defaults         *Defaults     `json:"defaults,omitempty"`
	Clips            []*Clip       `json:"clips"`
	AudioFilePath    string        `json:"audioFilePath,omitempty"`
	KeepSourceAudio  bool          `json:"keepSourceAudio,omitempty"`
	LoopAudio        bool          `json:"loopAudio,omitempty"`
	ClipsAudioVolume float64       `json:"clipsAudioVolume,omitempty"`
	OutputVolume     float64       `json:"outputVolume,omitempty"`
	AudioTracks      []*AudioTrack `json:"audioTracks,omitempty"`
	Verbose          bool          `json:"verbose,omitempty"`
	Fast             bool          `json:"fast,omitempty"`
}

// Defaults 定义默认参数
type Defaults struct {
	Duration   float64     `json:"duration,omitempty"`
	Transition *Transition `json:"transition,omitempty"`
}

// Clip 定义视频片段
type Clip struct {
	Duration   float64     `json:"duration,omitempty"`
	Transition *Transition `json:"transition,omitempty"`
	Layers     []*Layer    `json:"layers"`
}

// Transition 定义转场效果
type Transition struct {
	Duration float64 `json:"duration,omitempty"`
	Name     string  `json:"name,omitempty"`
}

// Layer 定义图层
type Layer struct {
	Type     string  `json:"type"` // "video", "image", "title", "audio"
	Path     string  `json:"path,omitempty"`
	Text     string  `json:"text,omitempty"`
	Color    string  `json:"color,omitempty"`
	FontPath string  `json:"fontPath,omitempty"`
	FontSize int     `json:"fontSize,omitempty"`
	Width    float64 `json:"width,omitempty"`
	Height   float64 `json:"height,omitempty"`
	Left     float64 `json:"left,omitempty"`
	Top      float64 `json:"top,omitempty"`
	Start    float64 `json:"start,omitempty"`
	Stop     float64 `json:"stop,omitempty"`
}

// AudioTrack 定义音频轨道
type AudioTrack struct {
	Path      string  `json:"path"`
	MixVolume float64 `json:"mixVolume,omitempty"`
	CutFrom   float64 `json:"cutFrom,omitempty"`
	CutTo     float64 `json:"cutTo,omitempty"`
	Start     float64 `json:"start,omitempty"`
}

// Editly 是对标editly的高级视频编辑器
type Editly struct {
	spec *EditSpec
}

// NewEditly 创建一个新的Editly实例
func NewEditly(spec *EditSpec) *Editly {
	// 设置默认值
	if spec.Width == 0 {
		spec.Width = 640
	}
	if spec.Fps == 0 {
		spec.Fps = 25
	}
	if spec.Defaults == nil {
		spec.Defaults = &Defaults{Duration: 4}
	} else if spec.Defaults.Duration == 0 {
		spec.Defaults.Duration = 4
	}
	if spec.Defaults.Transition == nil {
		spec.Defaults.Transition = &Transition{Duration: 0.5, Name: "fade"}
	} else {
		if spec.Defaults.Transition.Duration == 0 {
			spec.Defaults.Transition.Duration = 0.5
		}
		if spec.Defaults.Transition.Name == "" {
			spec.Defaults.Transition.Name = "fade"
		}
	}

	return &Editly{spec: spec}
}

// FromFile 从JSON文件加载编辑规范
func FromFile(filePath string) (*Editly, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	spec := &EditSpec{}
	if filepath.Ext(filePath) == ".json5" {
		// 简化处理，实际应该使用JSON5解析器
		return nil, fmt.Errorf("JSON5 not supported in this simplified implementation")
	}

	err = json.Unmarshal(data, spec)
	if err != nil {
		return nil, err
	}

	return NewEditly(spec), nil
}

// Edit 视频编辑主函数
func (e *Editly) Edit() error {
	if e.spec.Verbose {
		fmt.Printf("开始编辑视频: %s\n", e.spec.OutPath)
		fmt.Printf("尺寸: %dx%d, 帧率: %d\n", e.spec.Width, e.spec.Height, e.spec.Fps)
	}

	// 根据不同的图层类型处理视频
	var streams []*Stream

	for i, clip := range e.spec.Clips {
		if e.spec.Verbose {
			fmt.Printf("处理片段 %d/%d\n", i+1, len(e.spec.Clips))
		}

		clipDuration := clip.Duration
		if clipDuration == 0 && e.spec.Defaults != nil {
			clipDuration = e.spec.Defaults.Duration
		}

		// 如果片段没有指定持续时间，尝试从视频图层获取
		// 注意：实际实现应该使用Probe获取视频真实时长
		if clipDuration <= 0 {
			for _, layer := range clip.Layers {
				if layer.Type == "video" && layer.Path != "" {
					// 简化处理，实际应该解析视频文件获取真实时长
					// 这里保持默认值4秒作为演示
					clipDuration = 4.0
					break
				}
			}
		}

		// 确保至少有默认持续时间
		if clipDuration <= 0 {
			clipDuration = 4.0
		}

		clipStreams := make([]*Stream, 0)

		for _, layer := range clip.Layers {
			stream, err := e.processLayer(layer, clipDuration)
			if err != nil {
				return fmt.Errorf("处理图层时出错: %v", err)
			}
			if stream != nil {
				clipStreams = append(clipStreams, stream)
			}
		}

		// 合并同一片段中的多个图层
		if len(clipStreams) > 0 {
			var clipStream *Stream
			if len(clipStreams) == 1 {
				clipStream = clipStreams[0]
			} else {
				// 使用overlay合并多个图层
				clipStream = clipStreams[0]
				for i := 1; i < len(clipStreams); i++ {
					clipStream = clipStream.Overlay(clipStreams[i], "shortest")
				}
			}

			// 只对非视频图层设置时长
			hasVideoLayer := false
			for _, layer := range clip.Layers {
				if layer.Type == "video" {
					hasVideoLayer = true
					break
				}
			}

			// 如果片段中没有视频图层，则需要设置时长
			if !hasVideoLayer && clipDuration > 0 {
				clipStream = clipStream.Trim(KwArgs{"duration": clipDuration})
			}

			streams = append(streams, clipStream)
		}
	}

	// 连接所有片段
	if len(streams) == 0 {
		return fmt.Errorf("没有有效的视频片段")
	}

	var finalStream *Stream
	if len(streams) == 1 {
		finalStream = streams[0]
	} else {
		finalStream = Concat(streams)
	}

	// 添加背景音乐
	if e.spec.AudioFilePath != "" {
		audioStream := Input(e.spec.AudioFilePath)
		if e.spec.LoopAudio {
			audioStream = audioStream.Filter("loop", Args{"-1"})
		}
		finalStream = finalStream.Overlay(audioStream, "shortest")
	}

	// 设置输出参数
	kwargs := KwArgs{}
	if e.spec.Width > 0 && e.spec.Height > 0 {
		kwargs["s"] = fmt.Sprintf("%dx%d", e.spec.Width, e.spec.Height)
	}
	if e.spec.Fps > 0 {
		kwargs["r"] = e.spec.Fps
	}

	// 快速模式
	if e.spec.Fast {
		kwargs["preset"] = "ultrafast"
	}

	// 输出文件
	err := finalStream.Output(e.spec.OutPath, kwargs).
		OverWriteOutput().
		Run()

	if err != nil {
		return fmt.Errorf("输出视频时出错: %v", err)
	}

	if e.spec.Verbose {
		fmt.Printf("视频编辑完成: %s\n", e.spec.OutPath)
	}

	return nil
}

// processLayer 处理单个图层
func (e *Editly) processLayer(layer *Layer, duration float64) (*Stream, error) {
	// 确保持续时间有效
	if duration <= 0 {
		duration = 4.0 // 默认4秒
	}

	switch layer.Type {
	case "video":
		if layer.Path == "" {
			return nil, fmt.Errorf("视频图层缺少路径")
		}
		stream := Input(layer.Path)

		// 处理裁剪
		if layer.Start > 0 || layer.Stop > 0 {
			if layer.Stop > layer.Start {
				stream = stream.Trim(KwArgs{"start": layer.Start, "end": layer.Stop})
			}
		}

		// 调整尺寸
		if layer.Width > 0 && layer.Height > 0 {
			scale := fmt.Sprintf("scale=%d:%d", int(layer.Width), int(layer.Height))
			stream = stream.Filter("scale", Args{scale})
		}

		return stream, nil

	case "image":
		if layer.Path == "" {
			return nil, fmt.Errorf("图片图层缺少路径")
		}
		// 图片需要设置持续时间
		stream := Input(layer.Path, KwArgs{"loop": 1, "t": duration})

		return stream, nil

	case "title":
		// 简化处理 - 实际应该创建一个带文字的视频流
		// 这里只是示意，实际实现需要更复杂的处理
		if layer.Text == "" {
			return nil, fmt.Errorf("标题图层缺少文本")
		}

		// 创建纯色背景
		colorStr := layer.Color
		if colorStr == "" {
			colorStr = "black"
		}

		// 创建带文字的视频流需要更复杂的处理
		// 这里简化处理，实际应该使用drawtext过滤器
		width := e.spec.Width
		if width == 0 {
			width = 640
		}
		height := e.spec.Height
		if height == 0 {
			height = 480
		}

		// 创建一个纯色背景，持续指定时长
		stream := Input("color="+colorStr, KwArgs{"f": "lavfi", "t": duration}).
			Filter("scale", Args{fmt.Sprintf("%d:%d", width, height)})

		// 添加文字需要使用drawtext过滤器
		// 这里只是示意，实际实现需要正确设置字体等参数
		if layer.Text != "" {
			textArgs := KwArgs{
				"text": layer.Text,
			}
			if layer.FontPath != "" {
				textArgs["fontfile"] = layer.FontPath
			}
			if layer.FontSize > 0 {
				textArgs["fontsize"] = layer.FontSize
			} else {
				textArgs["fontsize"] = 24
			}

			// 简化处理，实际应该正确设置文字位置等
			stream = stream.Filter("drawtext", Args{}, textArgs)
		}

		return stream, nil

	case "audio":
		if layer.Path == "" {
			return nil, fmt.Errorf("音频图层缺少路径")
		}
		stream := Input(layer.Path)

		// 处理裁剪
		if layer.Start > 0 || layer.Stop > 0 {
			if layer.Stop > layer.Start {
				stream = stream.Trim(KwArgs{"start": layer.Start, "end": layer.Stop})
			}
		}

		return stream, nil

	case "fill-color":
		colorStr := layer.Color
		if colorStr == "" {
			colorStr = "black"
		}

		width := e.spec.Width
		if width == 0 {
			width = 640
		}
		height := e.spec.Height
		if height == 0 {
			height = 480
		}

		// 创建一个纯色背景，持续指定时长
		stream := Input("color="+colorStr, KwArgs{"f": "lavfi", "t": duration}).
			Filter("scale", Args{fmt.Sprintf("%d:%d", width, height)})

		return stream, nil

	default:
		return nil, fmt.Errorf("不支持的图层类型: %s", layer.Type)
	}
}

// parseColor 解析颜色字符串
func parseColor(colorStr string) (color.RGBA, error) {
	// 支持 "red", "blue" 等名称和 "#FF0000" 格式
	colorStr = strings.ToLower(colorStr)

	// 预定义颜色
	colors := map[string]color.RGBA{
		"black":   {0, 0, 0, 255},
		"white":   {255, 255, 255, 255},
		"red":     {255, 0, 0, 255},
		"green":   {0, 255, 0, 255},
		"blue":    {0, 0, 255, 255},
		"yellow":  {255, 255, 0, 255},
		"magenta": {255, 0, 255, 255},
		"cyan":    {0, 255, 255, 255},
	}

	if c, ok := colors[colorStr]; ok {
		return c, nil
	}

	// HEX颜色 (#RRGGBB)
	if strings.HasPrefix(colorStr, "#") && len(colorStr) == 7 {
		r, _ := strconv.ParseInt(colorStr[1:3], 16, 64)
		g, _ := strconv.ParseInt(colorStr[3:5], 16, 64)
		b, _ := strconv.ParseInt(colorStr[5:7], 16, 64)
		return color.RGBA{uint8(r), uint8(g), uint8(b), 255}, nil
	}

	return color.RGBA{0, 0, 0, 255}, fmt.Errorf("不支持的颜色格式: %s", colorStr)
}

// Edit 是一个便捷函数，直接编辑视频
func Edit(spec *EditSpec) error {
	editly := NewEditly(spec)
	return editly.Edit()
}
