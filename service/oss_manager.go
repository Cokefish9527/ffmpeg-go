package service

import (
	"fmt"
	"mime/multipart"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OSSManager 提供OSS存储管理功能
type OSSManager struct {
	Endpoint        string
	AccessKeyID     string
	AccessKeySecret string
	BucketName      string
	TsBucketName    string
	VideoOutputBucketName string // 视频编辑输出bucket
	ossService      *OSSService
	tsOssService    *OSSService
	videoOutputOssService *OSSService
}

// OSSConfig OSS配置信息
type OSSConfig struct {
	Endpoint        string `json:"endpoint"`
	AccessKeyID     string `json:"accessKeyId"`
	AccessKeySecret string `json:"accessKeySecret"`
	BucketName      string `json:"bucketName"`
	TsBucketName    string `json:"tsBucketName"`
	VideoOutputBucketName string `json:"videoOutputBucketName"` // 视频编辑输出bucket
}

// OSSObject OSS对象信息
// @Description OSS中存储的对象信息
type OSSObject struct {
	// 对象名称
	Name string `json:"name" example:"example.mp4"`
	// 对象大小(字节)
	Size int64 `json:"size" example:"1024000"`
	// 对象最后修改时间
	LastModified time.Time `json:"lastModified" example:"2023-08-10T10:30:00Z"`
	// 对象访问URL
	URL string `json:"url" example:"https://bucket-name.oss-cn-hangzhou.aliyuncs.com/example.mp4"`
}

// NewOSSManager 创建一个新的OSS管理器实例
func NewOSSManager(config OSSConfig) *OSSManager {
	ossManager := &OSSManager{
		Endpoint:        config.Endpoint,
		AccessKeyID:     config.AccessKeyID,
		AccessKeySecret: config.AccessKeySecret,
		BucketName:      config.BucketName,
		TsBucketName:    config.TsBucketName,
		VideoOutputBucketName: config.VideoOutputBucketName,
	}
	
	// 尝试初始化真实OSS服务
	ossService, err := NewOSSService(config.Endpoint, config.AccessKeyID, config.AccessKeySecret, config.BucketName)
	if err == nil && ossService != nil {
		ossManager.ossService = ossService
	}
	
	// 如果TS bucket名称不为空，初始化TS OSS服务
	if config.TsBucketName != "" {
		tsOssService, err := NewOSSService(config.Endpoint, config.AccessKeyID, config.AccessKeySecret, config.TsBucketName)
		if err == nil && tsOssService != nil {
			ossManager.tsOssService = tsOssService
		}
	}
	
	// 如果视频输出bucket名称不为空，初始化视频输出OSS服务
	if config.VideoOutputBucketName != "" {
		videoOutputOssService, err := NewOSSService(config.Endpoint, config.AccessKeyID, config.AccessKeySecret, config.VideoOutputBucketName)
		if err == nil && videoOutputOssService != nil {
			ossManager.videoOutputOssService = videoOutputOssService
		}
	}
	
	return ossManager
}

// UploadFile 上传文件到OSS
func (o *OSSManager) UploadFile(file multipart.File, header *multipart.FileHeader) (string, error) {
	// 必须使用真实的OSS服务，如果没有则返回错误
	if o.ossService == nil {
		return "", fmt.Errorf("OSS服务未初始化，请检查OSS配置")
	}
	
	return o.ossService.UploadFile(file, header)
}

// UploadFileToTsBucket 上传文件到TS OSS bucket
func (o *OSSManager) UploadFileToTsBucket(file multipart.File, header *multipart.FileHeader, path string) (string, error) {
    // 必须使用真实的TS OSS服务，如果没有则返回错误
    if o.tsOssService == nil {
        return "", fmt.Errorf("TS OSS服务未初始化，请检查TS OSS配置")
    }
    
    return o.tsOssService.UploadFileWithPath(file, header, path)
}

// UploadVideoOutput 上传视频编辑结果到输出bucket
func (o *OSSManager) UploadVideoOutput(file multipart.File, header *multipart.FileHeader, path string) (string, error) {
    // 必须使用真实的视频输出OSS服务，如果没有则返回错误
    if o.videoOutputOssService == nil {
        return "", fmt.Errorf("视频输出OSS服务未初始化，请检查视频输出OSS配置")
    }
    
    return o.videoOutputOssService.UploadFileWithPath(file, header, path)
}

// ExtractUserIDFromURL 从URL中提取用户ID
func (o *OSSManager) ExtractUserIDFromURL(fileURL string) (string, error) {
	parsedURL, err := url.Parse(fileURL)
	if err != nil {
		return "", fmt.Errorf("解析URL失败: %w", err)
	}
	
	// 路径格式一般为 /userid/filename
	path := parsedURL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	if len(parts) >= 1 {
		return parts[0], nil
	}
	
	return "", fmt.Errorf("无法从URL中提取用户ID: %s", fileURL)
}

// UploadFileWithPath 上传文件到OSS指定路径
func (o *OSSManager) UploadFileWithPath(file multipart.File, header *multipart.FileHeader, path string) (string, error) {
    // 必须使用真实的OSS服务，如果没有则返回错误
    if o.ossService == nil {
        return "", fmt.Errorf("OSS服务未初始化，请检查OSS配置")
    }
    
    return o.ossService.UploadFileWithPath(file, header, path)
}

// DownloadFile 从OSS下载文件
func (o *OSSManager) DownloadFile(objectName string, localFilePath string) error {
	// 如果有真实的OSS服务，则使用真实服务
	if o.ossService != nil {
		return o.ossService.DownloadFile(objectName, localFilePath)
	}
	
	// 否则使用模拟实现
	// 在实际实现中，这里会使用阿里云OSS SDK下载文件
	// 暂时返回模拟实现
	time.Sleep(100 * time.Millisecond)
	return nil
}

// ListObjects 列出存储空间中的对象
func (o *OSSManager) ListObjects(prefix string, maxKeys int) ([]OSSObject, error) {
	// 如果有真实的OSS服务，则使用真实服务
	if o.ossService != nil {
		return o.ossService.ListObjects(prefix, maxKeys)
	}
	
	// 否则使用模拟实现
	// 在实际实现中，这里会使用阿里云OSS SDK列出对象
	// 暂时返回模拟数据
	time.Sleep(100 * time.Millisecond)
	
	objects := []OSSObject{
		{
			Name:         "example1.mp4",
			Size:         1024000,
			LastModified: time.Now().Add(-2 * time.Hour),
			URL:          fmt.Sprintf("https://%s.%s/%s", o.BucketName, o.Endpoint, "example1.mp4"),
		},
		{
			Name:         "example2.mp4",
			Size:         2048000,
			LastModified: time.Now().Add(-1 * time.Hour),
			// 注意：实际的OSS endpoint格式可能不同
			URL: fmt.Sprintf("https://%s.%s/%s", o.BucketName, o.Endpoint, "example2.mp4"),
		},
	}
	
	return objects, nil
}

// DeleteObject 删除OSS中的对象
func (o *OSSManager) DeleteObject(objectName string) error {
	// 如果有真实的OSS服务，则使用真实服务
	if o.ossService != nil {
		return o.ossService.DeleteObject(objectName)
	}
	
	// 否则使用模拟实现
	// 在实际实现中，这里会使用阿里云OSS SDK删除对象
	// 暂时返回模拟实现
	time.Sleep(100 * time.Millisecond)
	return nil
}

// GetObjectURL 获取对象的访问URL
func (o *OSSManager) GetObjectURL(objectName string) string {
	// 如果有真实的OSS服务，则使用真实服务
	if o.ossService != nil {
		return o.ossService.GetObjectURL(objectName)
	}
	
	// 否则使用模拟实现
	// 使用与UploadFileWithPath一致的URL格式
	bucketName := o.BucketName
	if bucketName == "" {
		bucketName = "aima-hotvideogeneration-videolibrary"
	}
	
	endpoint := o.Endpoint
	if endpoint == "" {
		endpoint = "oss-cn-hangzhou.aliyuncs.com"
	}
	
	return fmt.Sprintf("https://%s.%s/%s", bucketName, endpoint, objectName)
}

// UploadFileToVideoOutputBucket 上传文件到视频输出桶
func (o *OSSManager) UploadFileToVideoOutputBucket(localFilePath, objectKey string) error {
    // 必须使用真实的视频输出OSS服务，如果没有则返回错误
    if o.videoOutputOssService == nil {
        return fmt.Errorf("视频输出OSS服务未初始化，请检查视频输出OSS配置")
    }
    
    // 打开本地文件
    file, err := os.Open(localFilePath)
    if err != nil {
        return fmt.Errorf("打开本地文件失败: %w", err)
    }
    defer file.Close()
    
    // 获取文件信息
    fileInfo, err := file.Stat()
    if err != nil {
        return fmt.Errorf("获取文件信息失败: %w", err)
    }
    
    // 创建multipart.FileHeader
    header := &multipart.FileHeader{
        Filename: filepath.Base(localFilePath),
        Size:     fileInfo.Size(),
    }
    
    // 上传文件
    _, err = o.videoOutputOssService.UploadFileWithPath(file, header, objectKey)
    if err != nil {
        return fmt.Errorf("上传文件到视频输出桶失败: %w", err)
    }
    
    return nil
}
