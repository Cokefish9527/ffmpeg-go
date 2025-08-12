package example

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/u2takey/ffmpeg-go/api"
)

// 模拟视频URL处理
func handleVideoURL(c *gin.Context) {
	var req struct {
		URL string `json:"url"`
	}
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": "无效请求"})
		return
	}
	
	// 模拟下载和处理视频
	fmt.Printf("处理视频URL: %s\n", req.URL)
	time.Sleep(500 * time.Millisecond)
	
	// 模拟返回结果
	result := map[string]interface{}{
		"status":     "success",
		"message":    "视频处理完成",
		"tsFilePath": "./sample_data/output.ts",
	}
	
	c.JSON(200, result)
}

func main() {
	// 创建测试服务器
	gin.SetMode(gin.TestMode)
	router := gin.New()
	
	// 注册路由
	router.POST("/video/url", handleVideoURL)
	
	// 测试URL列表
	testURLs := []string{
		"http://example.com/video1.mp4",
		"http://example.com/video2.mp4",
		"http://example.com/video3.mp4",
	}
	
	fmt.Println("=== 视频URL处理测试 ===")
	
	// 发送测试请求
	for i, url := range testURLs {
		// 构造请求
		requestBody := map[string]interface{}{
			"url": url,
		}
		
		jsonData, _ := json.Marshal(requestBody)
		req, _ := http.NewRequest("POST", "/video/url", bytes.NewBuffer(jsonData))
		req.Header.Set("Content-Type", "application/json")
		
		// 创建响应记录器
		w := httptest.NewRecorder()
		
		// 执行请求
		fmt.Printf("\n发送请求 %d: %s\n", i+1, url)
		router.ServeHTTP(w, req)
		
		// 读取响应
		responseBody, _ := io.ReadAll(w.Body)
		fmt.Printf("响应状态: %d\n", w.Code)
		fmt.Printf("响应内容: %s\n", string(responseBody))
	}
	
	fmt.Println("\n=== 视频URL处理测试完成 ===")
}