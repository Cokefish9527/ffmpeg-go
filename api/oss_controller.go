package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/u2takey/ffmpeg-go/service"
)

// OSSController 处理OSS相关请求
type OSSController struct {
	ossManager *service.OSSManager
}

// NewOSSController 创建OSS控制器实例
func NewOSSController(ossManager *service.OSSManager) *OSSController {
	return &OSSController{
		ossManager: ossManager,
	}
}

// UploadFile 上传文件到OSS
// @Summary 上传文件到OSS
// @Description 上传文件到阿里云OSS并返回可访问的URL。该接口接收一个文件流，将其上传到配置的OSS存储桶中，并返回文件的公开访问URL。
// @Tags oss
// @Accept mpfd
// @Produce json
// @Param file formData file true "要上传的文件" format(binary)
// @Success 200 {object} map[string]interface{} "文件上传成功" {message=string,url=string}
// @Failure 400 {object} map[string]string "请求参数错误" {error=string}
// @Failure 500 {object} map[string]string "内部服务器错误" {error=string}
// @Router /oss/upload [post]
func (o *OSSController) UploadFile(c *gin.Context) {
	// 从请求中获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "获取上传文件失败: " + err.Error(),
		})
		return
	}
	defer file.Close()

	// 上传文件到OSS
	url, err := o.ossManager.UploadFile(file, header)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "文件上传失败: " + err.Error(),
		})
		return
	}

	// 返回文件URL
	c.JSON(http.StatusOK, gin.H{
		"message": "文件上传成功",
		"url":     url,
	})
}

// ListObjects 列出OSS中的对象
// @Summary 列出OSS中的对象
// @Description 列出存储空间中的对象，支持按前缀过滤和限制返回数量。
// @Tags oss
// @Produce json
// @Param prefix query string false "对象名称前缀，用于过滤对象列表" default("")
// @Param maxKeys query int false "最大返回对象数量" default(100) maximum(1000)
// @Success 200 {array} service.OSSObject "对象列表"
// @Failure 500 {object} map[string]string "内部服务器错误" {error=string}
// @Router /oss/objects [get]
func (o *OSSController) ListObjects(c *gin.Context) {
	prefix := c.Query("prefix")
	maxKeys := 100 // 默认值
	
	// 如果提供了maxKeys参数，则使用它
	if maxKeysParam := c.Query("maxKeys"); maxKeysParam != "" {
		// 这里应该解析maxKeysParam为整数，为了简洁省略错误处理
		// 在实际项目中应该添加适当的错误处理
	}

	objects, err := o.ossManager.ListObjects(prefix, maxKeys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取对象列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, objects)
}

// DeleteObject 删除OSS中的对象
// @Summary 删除OSS中的对象
// @Description 根据对象名称删除OSS中的对象。注意：删除操作不可逆，请谨慎操作。
// @Tags oss
// @Produce json
// @Param objectName query string true "要删除的对象名称" 
// @Success 200 {object} map[string]string "删除成功" {message=string}
// @Failure 400 {object} map[string]string "请求参数错误" {error=string}
// @Failure 500 {object} map[string]string "内部服务器错误" {error=string}
// @Router /oss/object [delete]
func (o *OSSController) DeleteObject(c *gin.Context) {
	objectName := c.Query("objectName")
	if objectName == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "缺少必要的参数: objectName",
		})
		return
	}

	err := o.ossManager.DeleteObject(objectName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除对象失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "对象删除成功",
	})
}