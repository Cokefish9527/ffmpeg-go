package service

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"time"
)

// VideoInfo 视频信息结构
type VideoInfo struct {
	FileName   string  `json:"fileName"`
	FileSize   int64   `json:"fileSize"`
	Duration   float64 `json:"duration"`
	Codec      string  `json:"codec"`
	Width      int     `json:"width"`
	Height     int     `json:"height"`
	FPS        float64 `json:"fps"`
	Bitrate    int     `json:"bitrate"`
	AnalyzedAt time.Time `json:"analyzedAt"`
}

// VideoInfoCache 视频信息缓存
type VideoInfoCache struct {
	cache map[string]*VideoInfo
	mutex sync.RWMutex
}

// NewVideoInfoCache 创建新的视频信息缓存
func NewVideoInfoCache() *VideoInfoCache {
	return &VideoInfoCache{
		cache: make(map[string]*VideoInfo),
	}
}

// Get 获取视频信息
func (vic *VideoInfoCache) Get(filePath string) (*VideoInfo, bool) {
	vic.mutex.RLock()
	defer vic.mutex.RUnlock()
	
	info, exists := vic.cache[filePath]
	if !exists {
		return nil, false
	}
	
	// 检查缓存是否过期（超过1小时）
	if time.Since(info.AnalyzedAt) > time.Hour {
		return nil, false
	}
	
	// 检查文件是否被修改
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, false
	}
	
	if fileInfo.ModTime().After(info.AnalyzedAt) {
		return nil, false
	}
	
	return info, true
}

// Set 设置视频信息
func (vic *VideoInfoCache) Set(filePath string, info *VideoInfo) {
	vic.mutex.Lock()
	defer vic.mutex.Unlock()
	
	vic.cache[filePath] = info
}

// AnalyzeVideo 使用ffprobe分析视频信息
func (vic *VideoInfoCache) AnalyzeVideo(filePath string) (*VideoInfo, error) {
	// 检查缓存
	if info, exists := vic.Get(filePath); exists {
		return info, nil
	}
	
	// 检查文件是否存在
	fileInfo, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("无法获取文件信息: %w", err)
	}
	
	// 使用ffprobe获取视频信息
	cmd := exec.Command("ffprobe", "-v", "quiet", "-print_format", "json", "-show_format", "-show_streams", filePath)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("ffprobe执行失败: %w", err)
	}
	
	// 解析JSON输出
	var probeData map[string]interface{}
	if err := json.Unmarshal(output, &probeData); err != nil {
		return nil, fmt.Errorf("解析ffprobe输出失败: %w", err)
	}
	
	// 提取视频信息
	videoInfo := &VideoInfo{
		FileName:   filePath,
		FileSize:   fileInfo.Size(),
		AnalyzedAt: time.Now(),
	}
	
	// 获取时长
	if format, ok := probeData["format"].(map[string]interface{}); ok {
		if durationStr, ok := format["duration"].(string); ok {
			fmt.Sscanf(durationStr, "%f", &videoInfo.Duration)
		}
		
		if bitRateStr, ok := format["bit_rate"].(string); ok {
			fmt.Sscanf(bitRateStr, "%d", &videoInfo.Bitrate)
		}
	}
	
	// 获取视频流信息
	if streams, ok := probeData["streams"].([]interface{}); ok {
		for _, stream := range streams {
			if streamMap, ok := stream.(map[string]interface{}); ok {
				if codecType, ok := streamMap["codec_type"].(string); ok && codecType == "video" {
					// 获取编码
					if codecName, ok := streamMap["codec_name"].(string); ok {
						videoInfo.Codec = codecName
					}
					
					// 获取尺寸
					if width, ok := streamMap["width"].(float64); ok {
						videoInfo.Width = int(width)
					}
					if height, ok := streamMap["height"].(float64); ok {
						videoInfo.Height = int(height)
					}
					
					// 获取FPS
					if avgFrameRate, ok := streamMap["avg_frame_rate"].(string); ok {
						var num, den int
						if _, err := fmt.Sscanf(avgFrameRate, "%d/%d", &num, &den); err == nil && den != 0 {
							videoInfo.FPS = float64(num) / float64(den)
						}
					}
					break
				}
			}
		}
	}
	
	// 缓存结果
	vic.Set(filePath, videoInfo)
	
	return videoInfo, nil
}

// PreprocessInputFiles 预处理输入文件
func (vic *VideoInfoCache) PreprocessInputFiles(inputFiles []string, workDir string) ([]string, error) {
	processedFiles := make([]string, len(inputFiles))
	
	for i, file := range inputFiles {
		// 构造完整文件路径
		fullPath := filepath.Join(workDir, "video", file)
		
		// 分析视频信息
		_, err := vic.AnalyzeVideo(fullPath)
		if err != nil {
			return nil, fmt.Errorf("分析视频文件失败 %s: %w", file, err)
		}
		
		// 检查是否需要预处理
		// 如果视频编码不是h264，或者分辨率不匹配，或者fps不匹配，则需要预处理
		// 这里我们简化处理，只在必要时进行转码
		processedFiles[i] = fullPath
	}
	
	return processedFiles, nil
}