package main

import (
	"fmt"
	"io"
	"os"
	"time"

	"github.com/u2takey/ffmpeg-go/service"
)

func main() {
	fmt.Println("开始测试内存和缓存优化功能...")

	// 测试缓冲池
	testBufferPool()
	
	// 测试处理缓存
	testProcessingCache()
	
	// 测试文件复制性能
	testFileCopyPerformance()
}

func testBufferPool() {
	fmt.Println("\n=== 测试缓冲池 ===")
	
	// 测试缓冲区获取和归还
	bufferPool := service.GlobalBufferPool
	
	startTime := time.Now()
	for i := 0; i < 1000; i++ {
		// 获取不同大小的缓冲区
		size := 1024 + i%10000
		buf := bufferPool.Get(size)
		
		// 模拟使用缓冲区
		for j := 0; j < len(buf); j++ {
			buf[j] = byte(j % 256)
		}
		
		// 归还缓冲区
		bufferPool.Put(buf)
	}
	
	elapsed := time.Since(startTime)
	fmt.Printf("缓冲池测试完成，1000次操作耗时: %v\n", elapsed)
}

func testProcessingCache() {
	fmt.Println("\n=== 测试处理缓存 ===")
	
	cache := service.GlobalProcessingCache
	
	// 创建测试缓存条目
	key := "test_key_1"
	entry := &service.CacheEntry{
		OutputFile: "./test_output.mp4",
		CreatedAt:  time.Now(),
		Size:       1024 * 1024, // 1MB
	}
	
	// 添加到缓存
	cache.Put(key, entry)
	fmt.Printf("添加缓存条目: %s\n", key)
	
	// 查询缓存
	cachedEntry, exists := cache.Get(key)
	if exists {
		fmt.Printf("缓存命中: %s, 文件: %s, 大小: %d bytes\n", 
			key, cachedEntry.OutputFile, cachedEntry.Size)
	} else {
		fmt.Printf("缓存未命中: %s\n", key)
	}
	
	// 测试不存在的键
	_, exists = cache.Get("non_existent_key")
	if !exists {
		fmt.Printf("正确识别不存在的缓存键\n")
	}
	
	// 测试缓存存在性检查
	if cache.Exists(key) {
		fmt.Printf("缓存键 %s 存在\n", key)
	}
	
	if !cache.Exists("non_existent_key") {
		fmt.Printf("缓存键 non_existent_key 不存在\n")
	}
}

func testFileCopyPerformance() {
	fmt.Println("\n=== 测试文件复制性能 ===")
	
	// 获取当前工作目录
	wd, err := os.Getwd()
	if err != nil {
		fmt.Printf("无法获取当前工作目录: %v\n", err)
		return
	}
	
	// 选择一个测试文件（使用现有的视频文件）
	srcFile := wd + "/video/1.ts"
	dstFile := wd + "/temp/test_copy.mp4"
	
	// 确保目标目录存在
	os.MkdirAll(wd+"/temp", 0755)
	
	// 测试普通文件复制
	startTime := time.Now()
	err = copyFileRegular(srcFile, dstFile+"_regular")
	if err != nil {
		fmt.Printf("普通文件复制失败: %v\n", err)
		return
	}
	regularTime := time.Since(startTime)
	
	// 测试使用缓冲池的文件复制
	startTime = time.Now()
	err = copyFileWithBufferPool(srcFile, dstFile+"_bufferpool")
	if err != nil {
		fmt.Printf("缓冲池文件复制失败: %v\n", err)
		return
	}
	bufferPoolTime := time.Since(startTime)
	
	fmt.Printf("普通文件复制耗时: %v\n", regularTime)
	fmt.Printf("缓冲池文件复制耗时: %v\n", bufferPoolTime)
	
	// 清理测试文件
	os.Remove(dstFile + "_regular")
	os.Remove(dstFile + "_bufferpool")
	os.Remove(wd + "/temp")
}

// copyFileRegular 普通文件复制
func copyFileRegular(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = sourceFile.WriteTo(destFile)
	return err
}

// copyFileWithBufferPool 使用缓冲池复制文件
func copyFileWithBufferPool(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// 使用服务中的缓冲池
	buf := service.GlobalBufferPool.Get(64 * 1024) // 64KB缓冲区
	defer service.GlobalBufferPool.Put(buf)

	// 复制文件
	_, err = copyWithBuffer(sourceFile, destFile, buf)
	return err
}

// copyWithBuffer 使用指定缓冲区复制数据
func copyWithBuffer(src, dst *os.File, buf []byte) (int64, error) {
	var written int64
	for {
		nr, err := src.Read(buf)
		if nr > 0 {
			nw, err := dst.Write(buf[:nr])
			if nw > 0 {
				written += int64(nw)
			}
			if err != nil {
				return written, err
			}
			if nr != nw {
				return written, fmt.Errorf("写入不完整")
			}
		}
		if err != nil {
			if err == io.EOF {
				break
			}
			return written, err
		}
	}
	return written, nil
}