package utils

import (
	"os"
	"path/filepath"
	"sync"
)

var (
	globalLogger *Logger
	once         sync.Once
)

// InitGlobalLogger 初始化全局日志记录器
func InitGlobalLogger() {
	once.Do(func() {
		// 获取可执行文件目录
		execPath, err := os.Executable()
		if err != nil {
			execPath = "."
		}
		execDir := filepath.Dir(execPath)
		
		// 设置日志目录
		logDir := filepath.Join(execDir, "log")
		
		// 创建日志记录器
		logger, err := NewLogger(logDir, "ffmpeg_service", INFO, 10*1024*1024, 10) // 10MB每个文件，最多10个文件
		if err != nil {
			// 如果创建失败，使用标准错误输出
			logger = &Logger{
				level:  INFO,
				writer: os.Stderr,
			}
		}
		
		globalLogger = logger
	})
}

// GetGlobalLogger 获取全局日志记录器实例
func GetGlobalLogger() *Logger {
	if globalLogger == nil {
		InitGlobalLogger()
	}
	return globalLogger
}

// SetGlobalLoggerLevel 设置全局日志级别
func SetGlobalLoggerLevel(level LogLevel) {
	logger := GetGlobalLogger()
	logger.SetLevel(level)
}

// Debug 记录DEBUG级别日志
func Debug(message string, context map[string]string) {
	logger := GetGlobalLogger()
	logger.Debug(message, context)
}

// Info 记录INFO级别日志
func Info(message string, context map[string]string) {
	logger := GetGlobalLogger()
	logger.Info(message, context)
}

// Warn 记录WARN级别日志
func Warn(message string, context map[string]string) {
	logger := GetGlobalLogger()
	logger.Warn(message, context)
}

// Error 记录ERROR级别日志
func Error(message string, context map[string]string) {
	logger := GetGlobalLogger()
	logger.Error(message, context)
}

// Fatal 记录FATAL级别日志
func Fatal(message string, context map[string]string) {
	logger := GetGlobalLogger()
	logger.Fatal(message, context)
}