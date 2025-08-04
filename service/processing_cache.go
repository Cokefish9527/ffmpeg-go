package service

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// TaskCacheKey 处理任务缓存键
type TaskCacheKey struct {
	InputFiles []string `json:"input_files"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	FPS        int      `json:"fps"`
	Preset     string   `json:"preset"`
	// 可以根据需要添加更多参数
}

// GenerateKey 生成缓存键
func (tck *TaskCacheKey) GenerateKey() string {
	data, err := json.Marshal(tck)
	if err != nil {
		// 如果序列化失败，使用简单的方式生成键
		keyStr := fmt.Sprintf("%v_%d_%d_%d_%s", tck.InputFiles, tck.Width, tck.Height, tck.FPS, tck.Preset)
		hash := sha256.Sum256([]byte(keyStr))
		return hex.EncodeToString(hash[:])
	}
	
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

// CacheEntry 缓存条目
type CacheEntry struct {
	OutputFile string    `json:"output_file"`
	CreatedAt  time.Time `json:"created_at"`
	Size       int64     `json:"size"`
}

// ProcessingCache 处理结果缓存
type ProcessingCache struct {
	cache    map[string]*CacheEntry
	mutex    sync.RWMutex
	capacity int
}

// NewProcessingCache 创建新的处理结果缓存
func NewProcessingCache(capacity int) *ProcessingCache {
	return &ProcessingCache{
		cache:    make(map[string]*CacheEntry),
		capacity: capacity,
	}
}

// Get 获取缓存条目
func (pc *ProcessingCache) Get(key string) (*CacheEntry, bool) {
	pc.mutex.RLock()
	defer pc.mutex.RUnlock()
	
	entry, exists := pc.cache[key]
	if !exists {
		return nil, false
	}
	
	// 检查文件是否存在
	if _, err := os.Stat(entry.OutputFile); os.IsNotExist(err) {
		// 文件不存在，从缓存中删除
		go pc.remove(key)
		return nil, false
	}
	
	// 检查文件是否被修改
	fileInfo, err := os.Stat(entry.OutputFile)
	if err != nil {
		// 获取文件信息失败，从缓存中删除
		go pc.remove(key)
		return nil, false
	}
	
	if fileInfo.ModTime().Before(entry.CreatedAt) {
		// 文件比缓存条目旧，可能已被修改
		go pc.remove(key)
		return nil, false
	}
	
	return entry, true
}

// Put 添加缓存条目
func (pc *ProcessingCache) Put(key string, entry *CacheEntry) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	
	// 如果缓存已满，移除最旧的条目
	if len(pc.cache) >= pc.capacity {
		// 简单的实现：移除第一个条目
		// 实际项目中可以实现LRU等更复杂的淘汰策略
		for k := range pc.cache {
			delete(pc.cache, k)
			break
		}
	}
	
	pc.cache[key] = entry
}

// remove 删除缓存条目
func (pc *ProcessingCache) remove(key string) {
	pc.mutex.Lock()
	defer pc.mutex.Unlock()
	
	delete(pc.cache, key)
}

// Exists 检查缓存条目是否存在且有效
func (pc *ProcessingCache) Exists(key string) bool {
	_, exists := pc.Get(key)
	return exists
}

// GlobalProcessingCache 全局处理缓存实例
var GlobalProcessingCache = NewProcessingCache(100) // 默认容量100