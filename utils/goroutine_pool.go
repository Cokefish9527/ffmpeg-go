package utils

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Task 任务接口
type Task interface {
	Do() error
}

// FuncTask 函数任务实现
type FuncTask struct {
	f func() error
}

// Do 执行任务
func (ft *FuncTask) Do() error {
	if ft.f != nil {
		return ft.f()
	}
	return nil
}

// NewFuncTask 创建函数任务
func NewFuncTask(f func() error) *FuncTask {
	return &FuncTask{f: f}
}

// GoroutinePool Goroutine池结构
type GoroutinePool struct {
	// 配置
	minWorkers     int32         // 最小工作线程数
	maxWorkers     int32         // 最大工作线程数
	taskQueueSize  int           // 任务队列大小
	workerTimeout  time.Duration // 工作线程超时时间
	taskTimeout    time.Duration // 任务超时时间
	
	// 状态
	currentWorkers int32 // 当前工作线程数
	busyWorkers    int32 // 忙碌工作线程数
	totalTasks     int64 // 总任务数
	completedTasks int64 // 已完成任务数
	failedTasks    int64 // 失败任务数
	
	// 组件
	taskQueue chan Task       // 任务队列
	workers   []*worker       // 工作线程列表
	ctx       context.Context // 上下文
	cancel    context.CancelFunc
	wg        sync.WaitGroup
	mutex     sync.Mutex
}

// worker 工作线程结构
type worker struct {
	id         int32
	pool       *GoroutinePool
	taskChan   chan Task
	ctx        context.Context
	cancel     context.CancelFunc
	lastActive time.Time
}

// PoolStats Goroutine池统计信息
type PoolStats struct {
	MinWorkers     int32 `json:"minWorkers"`
	MaxWorkers     int32 `json:"maxWorkers"`
	CurrentWorkers int32 `json:"currentWorkers"`
	BusyWorkers    int32 `json:"busyWorkers"`
	TotalTasks     int64 `json:"totalTasks"`
	CompletedTasks int64 `json:"completedTasks"`
	FailedTasks    int64 `json:"failedTasks"`
	TaskQueueSize  int   `json:"taskQueueSize"`
}

// Option Goroutine池配置选项
type Option func(*GoroutinePool)

// WithMinWorkers 设置最小工作线程数
func WithMinWorkers(n int32) Option {
	return func(pool *GoroutinePool) {
		pool.minWorkers = n
	}
}

// WithMaxWorkers 设置最大工作线程数
func WithMaxWorkers(n int32) Option {
	return func(pool *GoroutinePool) {
		pool.maxWorkers = n
	}
}

// WithTaskQueueSize 设置任务队列大小
func WithTaskQueueSize(n int) Option {
	return func(pool *GoroutinePool) {
		pool.taskQueueSize = n
	}
}

// WithWorkerTimeout 设置工作线程超时时间
func WithWorkerTimeout(d time.Duration) Option {
	return func(pool *GoroutinePool) {
		pool.workerTimeout = d
	}
}

// WithTaskTimeout 设置任务超时时间
func WithTaskTimeout(d time.Duration) Option {
	return func(pool *GoroutinePool) {
		pool.taskTimeout = d
	}
}

// NewGoroutinePool 创建新的Goroutine池
func NewGoroutinePool(opts ...Option) *GoroutinePool {
	ctx, cancel := context.WithCancel(context.Background())
	
	pool := &GoroutinePool{
		minWorkers:    2,
		maxWorkers:    100,
		taskQueueSize: 1000,
		taskQueue:     make(chan Task, 1000),
		workers:       make([]*worker, 0),
		ctx:           ctx,
		cancel:        cancel,
		workerTimeout: time.Minute * 5,
		taskTimeout:   time.Minute * 10,
	}
	
	// 应用配置选项
	for _, opt := range opts {
		opt(pool)
	}
	
	// 确保最小工作线程数不小于1
	if pool.minWorkers < 1 {
		pool.minWorkers = 1
	}
	
	// 确保最大工作线程数不小于最小工作线程数
	if pool.maxWorkers < pool.minWorkers {
		pool.maxWorkers = pool.minWorkers
	}
	
	// 确保任务队列大小不小于1
	if pool.taskQueueSize < 1 {
		pool.taskQueueSize = 1
	}
	
	// 重新创建任务队列
	pool.taskQueue = make(chan Task, pool.taskQueueSize)
	
	// 初始化最小工作线程
	pool.initWorkers()
	
	return pool
}

// initWorkers 初始化工作线程
func (pool *GoroutinePool) initWorkers() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	for i := int32(0); i < pool.minWorkers; i++ {
		pool.createWorker()
	}
}

// createWorker 创建工作线程
func (pool *GoroutinePool) createWorker() {
	if atomic.LoadInt32(&pool.currentWorkers) >= pool.maxWorkers {
		return
	}
	
	workerCtx, workerCancel := context.WithCancel(pool.ctx)
	
	w := &worker{
		id:         atomic.AddInt32(&pool.currentWorkers, 1) - 1,
		pool:       pool,
		taskChan:   make(chan Task, 1),
		ctx:        workerCtx,
		cancel:     workerCancel,
		lastActive: time.Now(),
	}
	
	pool.workers = append(pool.workers, w)
	pool.wg.Add(1)
	
	go w.run()
}

// Submit 提交任务
func (pool *GoroutinePool) Submit(task Task) error {
	if task == nil {
		return fmt.Errorf("任务不能为空")
	}
	
	// 检查池是否已关闭
	select {
	case <-pool.ctx.Done():
		return fmt.Errorf("Goroutine池已关闭")
	default:
	}
	
	// 尝试提交任务
	select {
	case pool.taskQueue <- task:
		atomic.AddInt64(&pool.totalTasks, 1)
		// 检查是否需要创建新的工作线程
		pool.checkAndScale()
		return nil
	case <-pool.ctx.Done():
		return fmt.Errorf("Goroutine池已关闭")
	}
}

// SubmitFunc 提交函数任务
func (pool *GoroutinePool) SubmitFunc(f func() error) error {
	return pool.Submit(NewFuncTask(f))
}

// checkAndScale 检查并扩展工作线程
func (pool *GoroutinePool) checkAndScale() {
	currentWorkers := atomic.LoadInt32(&pool.currentWorkers)
	busyWorkers := atomic.LoadInt32(&pool.busyWorkers)
	
	// 如果任务队列中的任务数量大于忙碌工作线程数量，且当前工作线程数小于最大工作线程数，则创建新工作线程
	queueLen := int32(len(pool.taskQueue))
	if queueLen > busyWorkers && currentWorkers < pool.maxWorkers {
		pool.mutex.Lock()
		// 双重检查
		if int32(len(pool.taskQueue)) > atomic.LoadInt32(&pool.busyWorkers) && 
		   atomic.LoadInt32(&pool.currentWorkers) < pool.maxWorkers {
			pool.createWorker()
		}
		pool.mutex.Unlock()
	}
}

// dispatch 分发任务给工作线程
func (pool *GoroutinePool) dispatch() {
	for {
		select {
		case task := <-pool.taskQueue:
			// 分发任务给工作线程
			pool.dispatchToWorker(task)
		case <-pool.ctx.Done():
			return
		}
	}
}

// dispatchToWorker 分发任务给工作线程
func (pool *GoroutinePool) dispatchToWorker(task Task) {
	// 查找空闲的工作线程
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	var idleWorker *worker
	for _, w := range pool.workers {
		select {
		case <-w.ctx.Done():
			// 工作线程已关闭，跳过
			continue
		default:
		}
		
		// 检查工作线程是否空闲
		select {
		case w.taskChan <- task:
			idleWorker = w
			atomic.AddInt32(&pool.busyWorkers, 1)
			w.lastActive = time.Now()
			break
		default:
			// 工作线程忙碌，继续查找
		}
	}
	
	// 如果没有找到空闲的工作线程，创建新的工作线程
	if idleWorker == nil {
		if atomic.LoadInt32(&pool.currentWorkers) < pool.maxWorkers {
			pool.createWorker()
			// 重新尝试分发任务
			if len(pool.workers) > 0 {
				w := pool.workers[len(pool.workers)-1]
				select {
				case w.taskChan <- task:
					atomic.AddInt32(&pool.busyWorkers, 1)
					w.lastActive = time.Now()
				default:
					// 如果还是无法分发，将任务重新放回队列
					select {
					case pool.taskQueue <- task:
					default:
						// 如果队列已满，丢弃任务
						atomic.AddInt64(&pool.failedTasks, 1)
					}
				}
			}
		} else {
			// 达到最大工作线程数，将任务重新放回队列
			select {
			case pool.taskQueue <- task:
			default:
				// 如果队列已满，丢弃任务
				atomic.AddInt64(&pool.failedTasks, 1)
			}
		}
	}
}

// Start 启动Goroutine池
func (pool *GoroutinePool) Start() {
	go pool.dispatch()
	go pool.monitor()
	
	Info("Goroutine池启动", map[string]string{
		"minWorkers": fmt.Sprintf("%d", pool.minWorkers),
		"maxWorkers": fmt.Sprintf("%d", pool.maxWorkers),
		"taskQueueSize": fmt.Sprintf("%d", pool.taskQueueSize),
	})
}

// monitor 监控Goroutine池状态
func (pool *GoroutinePool) monitor() {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			pool.cleanupWorkers()
		case <-pool.ctx.Done():
			return
		}
	}
}

// cleanupWorkers 清理空闲的工作线程
func (pool *GoroutinePool) cleanupWorkers() {
	pool.mutex.Lock()
	defer pool.mutex.Unlock()
	
	currentWorkers := atomic.LoadInt32(&pool.currentWorkers)
	if currentWorkers <= pool.minWorkers {
		return
	}
	
	now := time.Now()
	var activeWorkers []*worker
	
	for _, w := range pool.workers {
		select {
		case <-w.ctx.Done():
			// 工作线程已关闭
			atomic.AddInt32(&pool.currentWorkers, -1)
			continue
		default:
		}
		
		// 检查工作线程是否空闲超时
		if now.Sub(w.lastActive) > pool.workerTimeout && 
		   currentWorkers > pool.minWorkers &&
		   len(w.taskChan) == 0 {
			// 关闭工作线程
			w.cancel()
			atomic.AddInt32(&pool.currentWorkers, -1)
			currentWorkers--
		} else {
			// 保留工作线程
			activeWorkers = append(activeWorkers, w)
		}
	}
	
	pool.workers = activeWorkers
}

// Stop 停止Goroutine池
func (pool *GoroutinePool) Stop() {
	Info("正在停止Goroutine池", nil)
	
	pool.cancel()
	pool.wg.Wait()
	
	close(pool.taskQueue)
	
	Info("Goroutine池已停止", map[string]string{
		"totalTasks": fmt.Sprintf("%d", atomic.LoadInt64(&pool.totalTasks)),
		"completedTasks": fmt.Sprintf("%d", atomic.LoadInt64(&pool.completedTasks)),
		"failedTasks": fmt.Sprintf("%d", atomic.LoadInt64(&pool.failedTasks)),
	})
}

// GetStats 获取池统计信息
func (pool *GoroutinePool) GetStats() *PoolStats {
	return &PoolStats{
		MinWorkers:     pool.minWorkers,
		MaxWorkers:     pool.maxWorkers,
		CurrentWorkers: atomic.LoadInt32(&pool.currentWorkers),
		BusyWorkers:    atomic.LoadInt32(&pool.busyWorkers),
		TotalTasks:     atomic.LoadInt64(&pool.totalTasks),
		CompletedTasks: atomic.LoadInt64(&pool.completedTasks),
		FailedTasks:    atomic.LoadInt64(&pool.failedTasks),
		TaskQueueSize:  len(pool.taskQueue),
	}
}

// run 工作线程运行方法
func (w *worker) run() {
	defer func() {
		w.pool.wg.Done()
		atomic.AddInt32(&w.pool.currentWorkers, -1)
		atomic.AddInt32(&w.pool.busyWorkers, -1)
		close(w.taskChan)
	}()
	
	for {
		select {
		case task := <-w.pool.taskQueue:
			// 执行任务
			w.executeTask(task)
			atomic.AddInt32(&w.pool.busyWorkers, -1)
		case task := <-w.taskChan:
			// 执行任务
			w.executeTask(task)
			atomic.AddInt32(&w.pool.busyWorkers, -1)
		case <-w.ctx.Done():
			return
		}
	}
}

// executeTask 执行任务
func (w *worker) executeTask(task Task) {
	defer func() {
		if r := recover(); r != nil {
			Error("任务执行发生panic", map[string]string{
				"workerId": fmt.Sprintf("%d", w.id),
				"panic":    fmt.Sprintf("%v", r),
			})
			atomic.AddInt64(&w.pool.failedTasks, 1)
		}
	}()
	
	// 创建带超时的上下文
	var ctx context.Context
	var cancel context.CancelFunc
	
	if w.pool.taskTimeout > 0 {
		ctx, cancel = context.WithTimeout(w.ctx, w.pool.taskTimeout)
		defer cancel()
	} else {
		ctx = w.ctx
	}
	
	// 在goroutine中执行任务，以便可以被取消
	done := make(chan error, 1)
	go func() {
		done <- task.Do()
	}()
	
	select {
	case err := <-done:
		if err != nil {
			Error("任务执行失败", map[string]string{
				"workerId": fmt.Sprintf("%d", w.id),
				"error":    err.Error(),
			})
			atomic.AddInt64(&w.pool.failedTasks, 1)
		} else {
			atomic.AddInt64(&w.pool.completedTasks, 1)
		}
	case <-ctx.Done():
		Error("任务执行超时", map[string]string{
			"workerId": fmt.Sprintf("%d", w.id),
			"timeout":  w.pool.taskTimeout.String(),
		})
		atomic.AddInt64(&w.pool.failedTasks, 1)
	}
}