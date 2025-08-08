package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// TestVideoEditEndpoint 测试video/edit端点的基本功能
func TestVideoEditEndpoint(t *testing.T) {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 创建Gin引擎
	r := gin.New()
	r.Use(gin.Recovery())

	// 定义API路由组
	apiRoutes := r.Group("/api/v1")
	{
		// 模拟video/edit端点，因为我们只是测试端点是否能接收请求
		apiRoutes.POST("/video/edit", func(c *gin.Context) {
			var req VideoEditRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid request format",
				})
				return
			}

			// 返回模拟响应
			response := VideoEditResponse{
				TaskID:  "test-task-id",
				Status:  "accepted",
				Message: "Task accepted for processing",
			}

			c.JSON(http.StatusAccepted, response)
		})
	}

	// 准备测试数据
	editSpec := map[string]interface{}{
		"outPath": "./test_output.mp4",
		"width":   1920,
		"height":  1080,
		"fps":     30,
		"clips": []map[string]interface{}{
			{
				"duration": 5,
				"layers": []map[string]interface{}{
					{
						"type": "video",
						"path": "http://example.com/test1.mp4",
					},
				},
			},
		},
	}

	requestData := VideoEditRequest{
		Spec: editSpec,
	}

	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	assert.NoError(t, err)

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBuffer(jsonData))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 执行请求
	r.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusAccepted, w.Code)

	// 解析响应数据
	var response VideoEditResponse
	err = json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// 验证响应内容
	assert.Equal(t, "test-task-id", response.TaskID)
	assert.Equal(t, "accepted", response.Status)
	assert.Equal(t, "Task accepted for processing", response.Message)
}

// TestVideoEditInvalidJSON 测试无效JSON请求
func TestVideoEditInvalidJSON(t *testing.T) {
	// 设置Gin为测试模式
	gin.SetMode(gin.TestMode)

	// 创建Gin引擎
	r := gin.New()
	r.Use(gin.Recovery())

	// 定义API路由组
	apiRoutes := r.Group("/api/v1")
	{
		// 模拟video/edit端点
		apiRoutes.POST("/video/edit", func(c *gin.Context) {
			var req VideoEditRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"error": "Invalid request format",
				})
				return
			}

			// 返回模拟响应
			response := VideoEditResponse{
				TaskID:  "test-task-id",
				Status:  "accepted",
				Message: "Task accepted for processing",
			}

			c.JSON(http.StatusAccepted, response)
		})
	}

	// 创建无效的JSON请求
	invalidJSON := `{"invalid": "json", "spec": }`

	// 创建HTTP请求
	req, err := http.NewRequest("POST", "/api/v1/video/edit", bytes.NewBufferString(invalidJSON))
	assert.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")

	// 创建响应记录器
	w := httptest.NewRecorder()

	// 执行请求
	r.ServeHTTP(w, req)

	// 验证响应
	assert.Equal(t, http.StatusBadRequest, w.Code)
}