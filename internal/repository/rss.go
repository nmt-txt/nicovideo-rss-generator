package repository

import (
	"bytes"
	"fmt"
	"hash/crc32"
	"io"
	"sync"
	"time"
)

// RSSRepository RSSフィードをメモリに保持する。
// 生成と読取りは別goroutineで行われるためMutexを使う必要があり、dataプロパティを直接読むことは許可されない。
type RSSRepository struct {
	data       []byte
	modifiedAt time.Time
	etag       string
	mu         sync.RWMutex
}

func NewRSSRepository() *RSSRepository {
	return &RSSRepository{
		data:       []byte{},
		modifiedAt: time.Now(),
	}
}

func (r *RSSRepository) Feed() io.ReadSeeker {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return bytes.NewReader(r.data)
}

func (r *RSSRepository) SetFeed(data []byte) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.data = data
	r.modifiedAt = time.Now()
	r.etag = fmt.Sprintf("W/%d-%x", len(data), crc32.ChecksumIEEE(data))
}

func (r *RSSRepository) ModifiedAt() time.Time {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.modifiedAt
}

func (r *RSSRepository) Etag() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.etag
}
