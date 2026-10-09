package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// rateLimit 按客户端 IP 限制 window 内最多 limit 次请求
func rateLimit(limit int, window time.Duration) gin.HandlerFunc {
	var mu sync.Mutex
	hits := make(map[string][]time.Time)
	lastSweep := time.Now()

	return func(c *gin.Context) {
		now := time.Now()
		cutoff := now.Add(-window)
		ip := c.ClientIP()

		mu.Lock()
		if now.Sub(lastSweep) > window { // 顺便清理不活跃的 IP
			for k, ts := range hits {
				if len(ts) == 0 || ts[len(ts)-1].Before(cutoff) {
					delete(hits, k)
				}
			}
			lastSweep = now
		}
		recent := hits[ip][:0]
		for _, t := range hits[ip] {
			if t.After(cutoff) {
				recent = append(recent, t)
			}
		}
		allowed := len(recent) < limit
		if allowed {
			recent = append(recent, now)
		}
		hits[ip] = recent
		mu.Unlock()

		if !allowed {
			failWith(c, http.StatusTooManyRequests, "请求过于频繁，请稍后再试", "")
			return
		}
		c.Next()
	}
}
