package system

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// MemoryItem 内存存储项
type MemoryItem struct {
	Value  string
	Expiry time.Time
}

// IsExpired 检查是否过期
func (m *MemoryItem) IsExpired() bool {
	return time.Now().After(m.Expiry)
}

// MemoryStore 内存存储实现
type MemoryStore struct {
	data  map[string]*MemoryItem
	mutex sync.RWMutex
}

// NewMemoryStore 创建新的内存存储
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		data: make(map[string]*MemoryItem),
	}
}

// Set 设置键值对，带过期时间
func (m *MemoryStore) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	var strValue string
	switch v := value.(type) {
	case string:
		strValue = v
	default:
		strValue = fmt.Sprintf("%v", v)
	}

	expiry := time.Now().Add(expiration)
	m.data[key] = &MemoryItem{
		Value:  strValue,
		Expiry: expiry,
	}

	// 启动清理goroutine
	go m.cleanup()

	return nil
}

// Get 获取值
func (m *MemoryStore) Get(ctx context.Context, key string) (string, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	item, exists := m.data[key]
	if !exists {
		return "", fmt.Errorf("key not found")
	}

	if item.IsExpired() {
		// 异步删除过期项
		go func() {
			m.mutex.Lock()
			delete(m.data, key)
			m.mutex.Unlock()
		}()
		return "", fmt.Errorf("key not found")
	}

	return item.Value, nil
}

// Del 删除键
func (m *MemoryStore) Del(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	delete(m.data, key)
	return nil
}

// Exists 检查键是否存在
func (m *MemoryStore) Exists(ctx context.Context, key string) (int64, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	item, exists := m.data[key]
	if !exists || item.IsExpired() {
		return 0, nil
	}

	return 1, nil
}

// Incr 递增
func (m *MemoryStore) Incr(ctx context.Context, key string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	item, exists := m.data[key]
	if !exists {
		m.data[key] = &MemoryItem{
			Value:  "1",
			Expiry: time.Now().Add(24 * time.Hour), // 默认24小时过期
		}
		return nil
	}

	if item.IsExpired() {
		item.Value = "1"
		item.Expiry = time.Now().Add(24 * time.Hour)
		return nil
	}

	// 简单的字符串递增（适用于计数器）
	item.Value = fmt.Sprintf("%d", parseInt(item.Value)+1)
	return nil
}

// PTTL 获取剩余生存时间（毫秒）
func (m *MemoryStore) PTTL(ctx context.Context, key string) (time.Duration, error) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	item, exists := m.data[key]
	if !exists {
		return -2, nil // key不存在
	}

	if item.IsExpired() {
		return -2, nil // key已过期
	}

	remaining := time.Until(item.Expiry)
	if remaining < 0 {
		return -1, nil // 已过期但未清理
	}

	return remaining, nil
}

// TxPipeline 事务管道（简化实现）
func (m *MemoryStore) TxPipeline() interface{} {
	return &MemoryPipeline{store: m}
}

// cleanup 清理过期项
func (m *MemoryStore) cleanup() {
	// 简单的清理逻辑：每分钟清理一次过期项
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.mutex.Lock()
		now := time.Now()
		for key, item := range m.data {
			if item.Expiry.Before(now) {
				delete(m.data, key)
			}
		}
		m.mutex.Unlock()
	}
}

// parseInt 简单的字符串转整数
func parseInt(s string) int {
	var result int
	for _, r := range s {
		if r >= '0' && r <= '9' {
			result = result*10 + int(r-'0')
		} else {
			break
		}
	}
	return result
}

// MemoryPipeline 内存事务管道（简化实现）
type MemoryPipeline struct {
	store     *MemoryStore
	operations []func()
}

// Set 添加设置操作到管道
func (p *MemoryPipeline) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) {
	p.operations = append(p.operations, func() {
		p.store.Set(ctx, key, value, expiration)
	})
}

// Exec 执行管道操作
func (p *MemoryPipeline) Exec(ctx context.Context) error {
	for _, op := range p.operations {
		op()
	}
	p.operations = nil // 清空操作
	return nil
}

// 全局内存存储实例
var memoryStore *MemoryStore
var memoryOnce sync.Once

// GetMemoryStore 获取全局内存存储实例（单例）
func GetMemoryStore() *MemoryStore {
	memoryOnce.Do(func() {
		memoryStore = NewMemoryStore()
	})
	return memoryStore
}