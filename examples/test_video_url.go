package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// VideoURLRequest 视频URL请求结构体
type VideoURLRequest struct {
	URL string `json:"url"`
}

// VideoURLResponse 视频URL响应结构体
type VideoURLResponse struct {
	Status     string `json:"status"`
	Message    string `json:"message"`
	TSFilePath string `json:"tsFilePath,omitempty"`
	Error      string `json:"error,omitempty"`
}

func main() {
	// 测试视频URL
	videoURL := "https://hsai-hz.oss-cn-hangzhou.aliyuncs.com/%E8%A7%86%E9%A2%91%E7%B4%A0%E6%9D%90/%E5%90%8C%E7%B1%BB%E5%9E%8B%E8%A7%86%E9%A2%91/Download%20%2812%29.mp4"
	
	// API端点 (使用8083端口)
	apiURL := "http://localhost:8083/api/v1/video/url"
	
	// 构造请求数据
	requestData := VideoURLRequest{
		URL: videoURL,
	}
	
	// 将请求数据转换为JSON
	jsonData, err := json.Marshal(requestData)
	if err != nil {
		fmt.Printf("错误：无法序列化请求数据: %v\n", err)
		return
	}
	
	fmt.Printf("正在发送请求到: %s\n", apiURL)
	fmt.Printf("处理视频URL: %s\n", videoURL)
	fmt.Println("请确保API服务器正在运行...")
	fmt.Println()
	
	// 创建HTTP客户端，设置超时时间
	client := &http.Client{
		Timeout: 60 * time.Second, // 设置较长的超时时间以处理大文件
	}
	
	// 创建POST请求
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Printf("错误：无法创建请求: %v\n", err)
		return
	}
	
	// 设置请求头
	req.Header.Set("Content-Type", "application/json")
	
	// 记录开始时间
	startTime := time.Now()
	
	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("错误：请求失败: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	// 记录结束时间
	endTime := time.Now()
	
	// 读取响应体
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Printf("错误：无法读取响应体: %v\n", err)
		return
	}
	
	fmt.Printf("响应状态码: %d\n", resp.StatusCode)
	fmt.Printf("处理耗时: %v\n", endTime.Sub(startTime))
	fmt.Printf("原始响应: %s\n", string(body))
	fmt.Println()
	
	// 解析响应
	var response VideoURLResponse
	if err := json.Unmarshal(body, &response); err != nil {
		fmt.Printf("错误：无法解析响应: %v\n", err)
		return
	}
	
	// 输出结果
	fmt.Printf("解析后的响应内容:\n")
	fmt.Printf("  Status: %s\n", response.Status)
	fmt.Printf("  Message: %s\n", response.Message)
	
	if response.TSFilePath != "" {
		fmt.Printf("  TS文件路径: %s\n", response.TSFilePath)
	}
	
	if response.Error != "" {
		fmt.Printf("  错误信息: %s\n", response.Error)
	}
}