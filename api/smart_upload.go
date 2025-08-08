package api

import (
	"fmt"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	ffmpeg_go "github.com/u2takey/ffmpeg-go"
	"github.com/u2takey/ffmpeg-go/service"
)

// SmartUploadRequest 智能上传请求参数
type SmartUploadRequest struct {
	UserID string `json:"userId" form:"userId"`
}

// SmartUploadResponse 智能上传响应
type SmartUploadResponse struct {
	Message string `json:"message"`
	URL     string `json:"url"`
}

// SmartUpload godoc
// @Summary 智能文件上传
// @Description 接收文件流，判断是否是视频文件，如果不是视频文件，直接上传到OSS，
// @Description 如果是视频文件，转换为TS格式后上传到OSS。文件会根据用户ID存放到专属目录中。
// @Tags video
// @Accept mpfd
// @Produce json
// @Param userId query string true "用户ID"
// @Param file formData file true "要上传的文件"
// @Success 200 {object} SmartUploadResponse "文件上传成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /video/smart-upload [post]
func SmartUpload(c *gin.Context, ossManager *service.OSSManager) {
	// 获取用户ID参数
	userID := c.Query("userId")
	if userID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少必要的参数: userId",
		})
		return
	}

	// 从请求中获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "获取上传文件失败: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 创建临时文件
	tempDir := os.TempDir()
	tempFileName := fmt.Sprintf("%s%s", uuid.New().String(), filepath.Ext(header.Filename))
	tempFilePath := filepath.Join(tempDir, tempFileName)

	tempFile, err := os.Create(tempFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "创建临时文件失败: " + err.Error(),
		})
		return
	}
	defer tempFile.Close()
	defer os.Remove(tempFilePath) // 清理临时文件

	// 将上传的文件内容复制到临时文件
	_, err = tempFile.ReadFrom(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存临时文件失败: " + err.Error(),
		})
		return
	}

	// 检查是否为视频文件
	isVideo, err := isVideoFile(tempFilePath)
	if err != nil {
		// 即使检测出错，也记录日志并继续处理
		fmt.Printf("检查文件类型时出错: %v\n", err)
	}
	
	// 打印调试信息
	fmt.Printf("文件路径: %s, 是否为视频文件: %t\n", tempFilePath, isVideo)

	// 根据文件类型进行处理
	var url string
	if isVideo {
		// 如果是视频文件，转换为TS格式
		fmt.Println("检测到视频文件，开始转换为TS格式")
		url, err = processVideoFile(tempFilePath, header.Filename, userID, ossManager)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "处理视频文件失败: " + err.Error(),
			})
			return
		}
		fmt.Println("视频文件转换完成")
	} else {
		// 如果不是视频文件，直接上传
		fmt.Println("检测到非视频文件，直接上传")
		// 重新打开文件以供上传
		tempFile, err := os.Open(tempFilePath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "打开临时文件失败: " + err.Error(),
			})
			return
		}
		defer tempFile.Close()

·		// 直接上传到OSS，使用用户ID作为目录
		url, err = ossManager.UploadFileWithPath(tempFile, header, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "文件上传失败: " + err.Error(),
			})
			return
		}
	}

	// 返回成功响应
	c.JSON(http.StatusOK, SmartUploadResponse{
		Message: "文件上传成功",
		URL:     url,
	})
}

// isVideoFile 检查文件是否为视频文件
func isVideoFile(filePath string) (bool, error) {
	// 使用ffprobe检查文件信息
	data, err := ffmpeg_go.Probe(filePath)
	if err != nil {
		// 如果ffprobe执行失败，可能不是视频文件
		fmt.Printf("ffprobe执行失败: %v\n", err)
		return false, nil
	}
	
	// 打印探测到的数据
	fmt.Printf("探测到的文件信息: %s\n", data)

	// 检查返回的数据中是否包含视频流
	// 使用更准确的判断方式，忽略引号类型差异
	hasVideoStream := strings.Contains(data, "\"codec_type\":\"video\"") || 
	                 strings.Contains(data, `"codec_type":"video"`) ||
	                 strings.Contains(data, "\"codec_type\": \"video\"") ||
	                 strings.Contains(data, `"codec_type": "video"`)
	
	// 打印检查结果
	fmt.Printf("是否包含视频流: %t\n", hasVideoStream)
	
	// 如果上述方法失败，尝试使用strings.Contains和TrimSpace方法
	if !hasVideoStream {
		lines := strings.Split(data, "\n")
		for _, line := range lines {
			// 去除空格后检查
			trimmedLine := strings.TrimSpace(line)
			if strings.Contains(trimmedLine, "\"codec_type\"") && 
			   strings.Contains(trimmedLine, "\"video\"") {
				hasVideoStream = true
				fmt.Printf("通过备用方法检测到视频流: %s\n", trimmedLine)
				break
			}
			if strings.Contains(trimmedLine, `"codec_type"`) && 
			   strings.Contains(trimmedLine, `"video"`) {
				hasVideoStream = true
				fmt.Printf("通过备用方法检测到视频流: %s\n", trimmedLine)
				break
			}
		}
	}
	
	return hasVideoStream, nil
}

// processVideoFile 处理视频文件，转换为TS格式并上传
func processVideoFile(inputPath, originalFilename, userID string, ossManager *service.OSSManager) (string, error) {
	fmt.Printf("开始处理视频文件: %s\n", inputPath)
	
	// 生成输出文件路径（TS格式）
	ext := filepath.Ext(originalFilename)
	nameWithoutExt := strings.TrimSuffix(originalFilename, ext)
	outputFilename := fmt.Sprintf("%s.ts", nameWithoutExt)
    
    // 修正outputPath的生成方式
    outputPath := filepath.Join(filepath.Dir(inputPath), fmt.Sprintf("%s%s", uuid.New().String(), ".ts"))
	
	fmt.Printf("输入文件: %s, 输出文件: %s\n", inputPath, outputPath)

	// 使用FFmpeg转换为TS格式
	err := ffmpeg_go.Input(inputPath).
		Output(outputPath, ffmpeg_go.KwArgs{"vcodec": "copy", "acodec": "copy", "f": "mpegts"}).
		OverWriteOutput().
		Run()
	if err != nil {
		return "", fmt.Errorf("视频转换失败: %w", err)
	}
	
	// 检查转换后的文件是否存在
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return "", fmt.Errorf("转换后的文件未生成: %w", err)
	}
	
	fmt.Printf("视频转换完成，文件路径: %s\n", outputPath)

	// 打开转换后的文件
	convertedFile, err := os.Open(outputPath)
	if err != nil {
		return "", fmt.Errorf("打开转换后的文件失败: %w", err)
	}
	defer convertedFile.Close()

	// 获取文件信息
	fileInfo, err := convertedFile.Stat()
	if err != nil {
		return "", fmt.Errorf("获取文件信息失败: %w", err)
	}

	// 创建multipart.FileHeader
	newHeader := &multipart.FileHeader{
		Filename: outputFilename,
		Size:     fileInfo.Size(),
	}

	// 上传到OSS，使用用户ID作为目录
	url, err := ossManager.UploadFileWithPath(convertedFile, newHeader, userID)
	if err != nil {
		return "", fmt.Errorf("上传文件到OSS失败: %w", err)
	}
	
	fmt.Printf("文件上传成功，URL: %s\n", url)

	return url, nil
}