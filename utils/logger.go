package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// LogLevel 日志级别类型
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String 返回日志级别的字符串表示
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// Logger 日志记录器结构
type Logger struct {
	level     LogLevel
	writer    *os.File
	mutex     sync.Mutex
	maxSize   int64
	maxFiles  int
	logDir    string
	logPrefix string
}

// LogEntry 日志条目结构
type LogEntry struct {
	Timestamp string            `json:"timestamp"`
	Level     string            `json:"level"`
	Message   string            `json:"message"`
	File      string            `json:"file,omitempty"`
	Line      int               `json:"line,omitempty"`
	Function  string            `json:"function,omitempty"`
	Context   map[string]string `json:"context,omitempty"`
}

// NewLogger 创建新的日志记录器
func NewLogger(logDir, logPrefix string, level LogLevel, maxSize int64, maxFiles int) (*Logger, error) {
	// 创建日志目录
	if err := os.MkdirAll(logDir, 0755); err != nil {
		return nil, fmt.Errorf("创建日志目录失败: %v", err)
	}

	// 打开日志文件
	logFile := filepath.Join(logDir, fmt.Sprintf("%s.log", logPrefix))
	file, err := os.OpenFile(logFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		return nil, fmt.Errorf("打开日志文件失败: %v", err)
	}

	return &Logger{
		level:     level,
		writer:    file,
		maxSize:   maxSize,
		maxFiles:  maxFiles,
		logDir:    logDir,
		logPrefix: logPrefix,
	}, nil
}

// SetLevel 设置日志级别
func (l *Logger) SetLevel(level LogLevel) {
	l.mutex.Lock()
	defer l.mutex.Unlock()
	l.level = level
}

// Debug 记录DEBUG级别日志
func (l *Logger) Debug(message string, context map[string]string) {
	if l.level <= DEBUG {
		l.log(DEBUG, message, context)
	}
}

// Info 记录INFO级别日志
func (l *Logger) Info(message string, context map[string]string) {
	if l.level <= INFO {
		l.log(INFO, message, context)
	}
}

// Warn 记录WARN级别日志
func (l *Logger) Warn(message string, context map[string]string) {
	if l.level <= WARN {
		l.log(WARN, message, context)
	}
}

// Error 记录ERROR级别日志
func (l *Logger) Error(message string, context map[string]string) {
	if l.level <= ERROR {
		l.log(ERROR, message, context)
	}
}

// Fatal 记录FATAL级别日志
func (l *Logger) Fatal(message string, context map[string]string) {
	if l.level <= FATAL {
		l.log(FATAL, message, context)
	}
}

// log 记录日志的内部方法
func (l *Logger) log(level LogLevel, message string, context map[string]string) {
	// 获取调用者信息
	_, file, line, ok := runtime.Caller(2)
	if !ok {
		file = "unknown"
		line = 0
	} else {
		// 只保留文件名
		file = filepath.Base(file)
	}

	// 获取函数名
	pc, _, _, ok := runtime.Caller(2)
	var funcName string
	if ok {
		funcName = runtime.FuncForPC(pc).Name()
		// 只保留函数名，去掉包路径
		if idx := len(funcName) - 1; idx >= 0 {
			for i := idx; i >= 0; i-- {
				if funcName[i] == '/' {
					funcName = funcName[i+1:]
					break
				}
			}
		}
		if idx := len(funcName) - 1; idx >= 0 {
			for i := idx; i >= 0; i-- {
				if funcName[i] == '.' {
					funcName = funcName[i+1:]
					break
				}
			}
		}
	} else {
		funcName = "unknown"
	}

	// 创建日志条目
	entry := &LogEntry{
		Timestamp: time.Now().Format(time.RFC3339),
		Level:     level.String(),
		Message:   message,
		File:      file,
		Line:      line,
		Function:  funcName,
		Context:   context,
	}

	// 序列化为JSON
	data, err := json.Marshal(entry)
	if err != nil {
		// 如果JSON序列化失败，使用简单格式
		fallback := fmt.Sprintf("%s [%s] %s (%s:%d %s)\n", 
			entry.Timestamp, entry.Level, entry.Message, entry.File, entry.Line, entry.Function)
		l.write([]byte(fallback))
		return
	}

	// 添加换行符并写入
	data = append(data, '\n')
	l.write(data)
}

// write 写入日志数据
func (l *Logger) write(data []byte) {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	// 检查是否需要轮转日志
	if l.needRotate() {
		l.rotate()
	}

	// 写入日志
	l.writer.Write(data)
}

// needRotate 检查是否需要轮转日志
func (l *Logger) needRotate() bool {
	if l.maxSize <= 0 {
		return false
	}

	info, err := l.writer.Stat()
	if err != nil {
		return false
	}

	return info.Size() >= l.maxSize
}

// rotate 轮转日志文件
func (l *Logger) rotate() {
	// 关闭当前文件
	l.writer.Close()

	// 重命名当前日志文件
	currentFile := filepath.Join(l.logDir, fmt.Sprintf("%s.log", l.logPrefix))
	rotateFile := filepath.Join(l.logDir, fmt.Sprintf("%s_%s.log", l.logPrefix, time.Now().Format("20060102_150405")))

	// 重命名文件
	os.Rename(currentFile, rotateFile)

	// 删除多余的日志文件
	l.cleanup()

	// 创建新的日志文件
	file, err := os.OpenFile(currentFile, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	if err != nil {
		// 如果创建失败，使用标准错误输出
		l.writer = os.Stderr
		return
	}

	l.writer = file
}

// cleanup 清理多余的日志文件
func (l *Logger) cleanup() {
	if l.maxFiles <= 0 {
		return
	}

	// 查找匹配的日志文件
	pattern := filepath.Join(l.logDir, fmt.Sprintf("%s_*.log", l.logPrefix))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return
	}

	// 如果文件数量超过限制，删除最旧的文件
	if len(matches) > l.maxFiles {
		// 按文件名排序（实际上是按时间排序）
		// 这里简化处理，实际项目中应该使用更精确的排序方法
		for i := 0; i < len(matches)-l.maxFiles; i++ {
			os.Remove(matches[i])
		}
	}
}

// Close 关闭日志记录器
func (l *Logger) Close() error {
	l.mutex.Lock()
	defer l.mutex.Unlock()

	return l.writer.Close()
}