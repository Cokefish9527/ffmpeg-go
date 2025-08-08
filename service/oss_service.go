package service

import (
	"fmt"
	"mime/multipart"
	"path/filepath"

	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	"github.com/google/uuid"
)

// OSSService 提供基于阿里云OSS的实际服务实现
type OSSService struct {
	client     *oss.Client
	bucketName string
	bucket     *oss.Bucket
}

// NewOSSService 创建一个新的OSS服务实例
func NewOSSService(endpoint, accessKeyID, accessKeySecret, bucketName string) (*OSSService, error) {
	// 如果配置为空，则返回nil，使用模拟实现
	if endpoint == "" || accessKeyID == "" || accessKeySecret == "" || bucketName == "" {
		return nil, nil
	}

	// 创建OSS客户端
	client, err := oss.New(endpoint, accessKeyID, accessKeySecret)
	if err != nil {
		return nil, fmt.Errorf("创建OSS客户端失败: %w", err)
	}

	// 获取存储空间
	bucket, err := client.Bucket(bucketName)
	if err != nil {
		return nil, fmt.Errorf("获取存储空间失败: %w", err)
	}

	return &OSSService{
		client:     client,
		bucketName: bucketName,
		bucket:     bucket,
	}, nil
}

// UploadFile 上传文件到OSS
func (o *OSSService) UploadFile(file multipart.File, header *multipart.FileHeader) (string, error) {
    return o.UploadFileWithPath(file, header, "")
}

// UploadFileWithPath 上传文件到OSS指定路径
func (o *OSSService) UploadFileWithPath(file multipart.File, header *multipart.FileHeader, path string) (string, error) {
    // 生成唯一文件名，避免文件名重复
    fileExt := filepath.Ext(header.Filename)
    uniqueFileName := fmt.Sprintf("%s%s", uuid.New().String(), fileExt)
    
    // 构造对象Key，包含路径和唯一文件名
    objectKey := uniqueFileName
    if path != "" {
        // 确保路径以'/'结尾
        if path[len(path)-1] != '/' {
            path += "/"
        }
        objectKey = path + uniqueFileName
    }

    // 上传到OSS
    err := o.bucket.PutObject(objectKey, file)
    if err != nil {
        return "", fmt.Errorf("上传文件到OSS失败: %w", err)
    }

    // 构造文件URL（带签名，确保可以直接访问）
    // 签名有效期为1小时
    url, err := o.bucket.SignURL(objectKey, oss.HTTPGet, 3600)
    if err != nil {
        // 如果签名失败，返回默认URL
        return fmt.Sprintf("https://%s.%s/%s", o.bucketName, o.client.Config.Endpoint, objectKey), nil
    }
    
    return url, nil
}

// DownloadFile 从OSS下载文件
func (o *OSSService) DownloadFile(objectName string, localFilePath string) error {
	// 下载文件
	err := o.bucket.GetObjectToFile(objectName, localFilePath)
	if err != nil {
		return fmt.Errorf("从OSS下载文件失败: %w", err)
	}
	return nil
}

// ListObjects 列出存储空间中的对象
func (o *OSSService) ListObjects(prefix string, maxKeys int) ([]OSSObject, error) {
	// 设置列举选项
	options := []oss.Option{
		oss.MaxKeys(maxKeys),
	}
	
	if prefix != "" {
		options = append(options, oss.Prefix(prefix))
	}

	// 列举对象
	lor, err := o.bucket.ListObjects(options...)
	if err != nil {
		return nil, fmt.Errorf("列举OSS对象失败: %w", err)
	}

	// 转换为统一格式
	objects := make([]OSSObject, len(lor.Objects))
	for i, obj := range lor.Objects {
		objects[i] = OSSObject{
			Name:         obj.Key,
			Size:         obj.Size,
			LastModified: obj.LastModified,
			URL:          fmt.Sprintf("https://%s.%s/%s", o.bucketName, o.client.Config.Endpoint, obj.Key),
		}
	}

	return objects, nil
}

// DeleteObject 删除OSS中的对象
func (o *OSSService) DeleteObject(objectName string) error {
	// 删除对象
	err := o.bucket.DeleteObject(objectName)
	if err != nil {
		return fmt.Errorf("删除OSS对象失败: %w", err)
	}
	return nil
}

// GetObjectURL 获取对象的访问URL（带签名，确保可以直接访问）
func (o *OSSService) GetObjectURL(objectName string) string {
    // 生成带签名的URL，有效期为1小时
    url, err := o.bucket.SignURL(objectName, oss.HTTPGet, 3600)
    if err != nil {
        // 如果签名失败，返回默认URL
        return fmt.Sprintf("https://%s.%s/%s", o.bucketName, o.client.Config.Endpoint, objectName)
    }
    return url
}
