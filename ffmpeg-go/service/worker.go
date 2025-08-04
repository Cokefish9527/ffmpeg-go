package service

// Worker 工作线程结构
type Worker struct {
    // ... 其他字段 ...
    isActive bool // 添加一个isActive字段来表示工作线程是否活跃
}

// IsActive 检查工作线程是否活跃
func (w *Worker) IsActive() bool {
    return w.isActive
}