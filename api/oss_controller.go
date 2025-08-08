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

// UploadFile godoc
// @Summary 上传文件到OSS
// @Description 上传文件到阿里云OSS并返回可访问的URL
// @Tags oss
// @Accept mpfd
// @Produce json
// @Param file formData file true "要上传的文件"
// @Success 200 {object} map[string]string "文件上传成功及URL"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "内部服务器错误"
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

// ListObjects godoc
// @Summary 列出OSS中的对象
// @Description 列出存储空间中的对象
// @Tags oss
// @Produce json
// @Param prefix query string false "对象名称前缀"
// @Param maxKeys query int false "最大返回对象数量"
// @Success 200 {object} []service.OSSObject "对象列表"
// @Failure 500 {object} map[string]string "内部服务器错误"
// @Router /oss/objects [get]
func (o *OSSController) ListObjects(c *gin.Context) {
	prefix := c.Query("prefix")
	maxKeys := 100 // 默认值

	objects, err := o.ossManager.ListObjects(prefix, maxKeys)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取对象列表失败: " + err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, objects)
}

// DeleteObject godoc
// @Summary 删除OSS中的对象
// @Description 根据对象名称删除OSS中的对象
// @Tags oss
// @Produce json
// @Param objectName query string true "对象名称"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求参数错误"
// @Failure 500 {object} map[string]string "内部服务器错误"
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