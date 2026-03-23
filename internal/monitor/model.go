package monitor

type CacheStats struct {
	CacheName string  `json:"cacheName"`
	Size      int64   `json:"size"`
	HitCount  int64   `json:"hitCount"`
	MissCount int64   `json:"missCount"`
	HitRate   float64 `json:"hitRate"`
}
