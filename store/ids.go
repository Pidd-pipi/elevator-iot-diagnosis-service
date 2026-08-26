// Package store 提供内存仓储与 JSON 文件持久化。
package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sync/atomic"
	"time"
)

var seq atomic.Int64

// NewID 生成带业务前缀的全局唯一 ID，形如 event-1710000000123-a1b2c3...。
func NewID(prefix string) string {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		// crypto/rand 失败时退化为时间+序号，保证不阻塞业务。
		n := seq.Add(1)
		return fmt.Sprintf("%s-%d-%d", prefix, time.Now().UnixNano(), n)
	}
	return fmt.Sprintf("%s-%d-%s", prefix, time.Now().UnixMilli(), hex.EncodeToString(buf))
}
