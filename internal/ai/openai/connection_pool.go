package openai

import (
	"sync"
	"sync/atomic"
	"time"
)

// PooledConnection 连接池中的连接
type PooledConnection struct {
	ID         int
	LastUsed   time.Time
	Active     bool
	CreateTime time.Time
}

// ConnectionPool 连接池
type ConnectionPool struct {
	maxSize    int
	connections map[int]*PooledConnection
	nextID     int32
	active     int32
	mu         sync.RWMutex
}

// NewConnectionPool 创建连接池
func NewConnectionPool(maxSize int) *ConnectionPool {
	pool := &ConnectionPool{
		maxSize:    maxSize,
		connections: make(map[int]*PooledConnection),
	}

	// 预创建连接
	for i := 0; i < maxSize; i++ {
		pool.connections[i] = &PooledConnection{
			ID:         i,
			CreateTime: time.Now(),
			LastUsed:   time.Now(),
			Active:     false,
		}
	}

	return pool
}

// Get 获取连接
func (p *ConnectionPool) Get() *PooledConnection {
	p.mu.Lock()
	defer p.mu.Unlock()

	for _, conn := range p.connections {
		if !conn.Active {
			conn.Active = true
			conn.LastUsed = time.Now()
			atomic.AddInt32(&p.active, 1)
			return conn
		}
	}

	// 如果没有可用连接，创建新的
	newID := int(atomic.AddInt32(&p.nextID, 1))
	conn := &PooledConnection{
		ID:         newID,
		CreateTime: time.Now(),
		LastUsed:   time.Now(),
		Active:     true,
	}
	p.connections[newID] = conn
	atomic.AddInt32(&p.active, 1)

	return conn
}

// Put 归还连接
func (p *ConnectionPool) Put(conn *PooledConnection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if c, exists := p.connections[conn.ID]; exists {
		c.Active = false
		c.LastUsed = time.Now()
		atomic.AddInt32(&p.active, -1)
	}
}

// Stats 获取连接池统计
func (p *ConnectionPool) Stats() PoolStats {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return PoolStats{
		TotalConnections: len(p.connections),
		ActiveConnections: int(atomic.LoadInt32(&p.active)),
		MaxSize:          p.maxSize,
	}
}

// PoolStats 连接池统计
type PoolStats struct {
	TotalConnections   int `json:"total_connections"`
	ActiveConnections  int `json:"active_connections"`
	MaxSize            int `json:"max_size"`
}

// Cleanup 清理空闲连接
func (p *ConnectionPool) Cleanup(idleTimeout time.Duration) {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	for id, conn := range p.connections {
		if !conn.Active && now.Sub(conn.LastUsed) > idleTimeout {
			if len(p.connections) > p.maxSize {
				delete(p.connections, id)
			}
		}
	}
}
