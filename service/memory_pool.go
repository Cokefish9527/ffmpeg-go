package service

import (
	"sync"
)

// FramePool 视频帧内存池
type FramePool struct {
	pool sync.Pool
}

// VideoFrame 视频帧结构
type VideoFrame struct {
	Data     []byte
	Width    int
	Height   int
	Channels int
}

// Reset 重置视频帧
func (vf *VideoFrame) Reset() {
	// 重置数据但保留底层数组
	if vf.Data != nil {
		for i := range vf.Data {
			vf.Data[i] = 0
		}
	}
	vf.Width = 0
	vf.Height = 0
	vf.Channels = 0
}

// NewFramePool 创建新的视频帧内存池
func NewFramePool() *FramePool {
	return &FramePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &VideoFrame{}
			},
		},
	}
}

// Get 获取视频帧
func (fp *FramePool) Get() *VideoFrame {
	frame, ok := fp.pool.Get().(*VideoFrame)
	if !ok {
		return &VideoFrame{}
	}
	return frame
}

// Put 归还视频帧
func (fp *FramePool) Put(frame *VideoFrame) {
	frame.Reset()
	fp.pool.Put(frame)
}

// BufferPool 缓冲区内存池
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool 创建新的缓冲区内存池
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, 1024) // 默认大小1KB
			},
		},
	}
}

// Get 获取缓冲区
func (bp *BufferPool) Get(size int) []byte {
	buf, ok := bp.pool.Get().([]byte)
	if !ok || cap(buf) < size {
		return make([]byte, size)
	}
	return buf[:size]
}

// Put 归还缓冲区
func (bp *BufferPool) Put(buf []byte) {
	// 重置缓冲区但保留底层数组
	if buf != nil {
		buf = buf[:0]
		bp.pool.Put(buf[:cap(buf)])
	}
}

// GlobalBufferPool 全局缓冲池实例
var GlobalBufferPool = NewBufferPool()

// GlobalFramePool 全局帧池实例
var GlobalFramePool = NewFramePool()