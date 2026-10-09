package jwclient

import (
	"net/http"
	"net/url"
	"sync"
)

// Jar 按 host 保存 cookie 的简易 CookieJar，支持导出/导入以便持久化
type Jar struct {
	mu      sync.Mutex
	cookies map[string][]*http.Cookie
	dirty   bool // 自上次 TakeDirty 以来是否有新 cookie
}

// SetCookies 实现 http.CookieJar；同名 cookie 以新的为准
func (j *Jar) SetCookies(u *url.URL, cookies []*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	if j.cookies == nil {
		j.cookies = make(map[string][]*http.Cookie)
	}
	byName := make(map[string]*http.Cookie)
	for _, c := range j.cookies[u.Host] {
		byName[c.Name] = c
	}
	for _, c := range cookies {
		byName[c.Name] = c
	}
	merged := make([]*http.Cookie, 0, len(byName))
	for _, c := range byName {
		merged = append(merged, c)
	}
	j.cookies[u.Host] = merged
	j.dirty = true
}

// Cookies 实现 http.CookieJar
func (j *Jar) Cookies(u *url.URL) []*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	return j.cookies[u.Host]
}

// Export 导出全部 cookie
func (j *Jar) Export() map[string][]*http.Cookie {
	j.mu.Lock()
	defer j.mu.Unlock()
	out := make(map[string][]*http.Cookie, len(j.cookies))
	for host, list := range j.cookies {
		out[host] = append([]*http.Cookie(nil), list...)
	}
	return out
}

// Import 导入持久化的 cookie
func (j *Jar) Import(cookies map[string][]*http.Cookie) {
	j.mu.Lock()
	defer j.mu.Unlock()
	j.cookies = cookies
	j.dirty = false
}

// TakeDirty 返回是否有未持久化的变化，并清除标记
func (j *Jar) TakeDirty() bool {
	j.mu.Lock()
	defer j.mu.Unlock()
	d := j.dirty
	j.dirty = false
	return d
}
