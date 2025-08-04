// GetActiveWorkerCount 获取活跃Worker数量
func (wp *WorkerPool) GetActiveWorkerCount() int {
    wp.mutex.Lock()
    defer wp.mutex.Unlock()
    
    activeCount := 0
    for _, worker := range wp.workers {
        if worker.IsActive() { // 使用IsActive方法来判断是否活跃
            activeCount++
        }
    }
    return activeCount
}