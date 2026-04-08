package httpServer

import (
	"encoding/json"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
	"wssgo/config"
)

type responsePayload struct {
	ErrCode      int    `json:"errcode"`
	ResponseData string `json:"response_data"`
}

type visitorWindow struct {
	windowStart int64
	count       int64
}

type fixedWindowLimiter struct {
	mutex         sync.Mutex
	windowSeconds int64
	maxRequests   int64
	visitors      map[string]*visitorWindow
}

func newFixedWindowLimiter(windowSeconds, maxRequests int64) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		windowSeconds: windowSeconds,
		maxRequests:   maxRequests,
		visitors:      make(map[string]*visitorWindow),
	}
}

func (l *fixedWindowLimiter) Allow(key string, now time.Time) bool {
	if key == "" {
		key = "unknown"
	}
	windowStart := now.Unix() / l.windowSeconds

	l.mutex.Lock()
	defer l.mutex.Unlock()

	v, ok := l.visitors[key]
	if !ok || v.windowStart != windowStart {
		l.visitors[key] = &visitorWindow{
			windowStart: windowStart,
			count:       1,
		}
		return true
	}
	if v.count >= l.maxRequests {
		return false
	}
	v.count++
	return true
}

func writeJSON(w http.ResponseWriter, statusCode int, payload *responsePayload) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	body, err := json.Marshal(payload)
	if err != nil {
		_, _ = w.Write([]byte(`{"errcode":5000,"response_data":"marshal failed"}`))
		return
	}
	_, _ = w.Write(body)
}

func getClientKey(r *http.Request, keyBy string) string {
	switch strings.ToLower(keyBy) {
	case "ip":
		fallthrough
	default:
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			return r.RemoteAddr
		}
		return host
	}
}

func wrapWithBodyLimit(next http.Handler, maxBodyBytes int64) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		next.ServeHTTP(w, r)
	})
}

func wrapWithRateLimit(next http.Handler) http.Handler {
	rlConf := config.ServiceConf.HttpConf.RateLimit
	if !rlConf.Enabled {
		return next
	}

	if strings.ToLower(rlConf.Strategy) != "fixed_window" {
		log.Printf("unsupported rate limit strategy: %s, fallback to fixed_window", rlConf.Strategy)
	}
	limiter := newFixedWindowLimiter(rlConf.WindowSeconds, rlConf.MaxRequests)
	mode := strings.ToLower(rlConf.Mode)
	if mode != "enforce" && mode != "dry_run" {
		mode = "enforce"
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		key := getClientKey(r, rlConf.KeyBy)
		allowed := limiter.Allow(key, time.Now())
		if allowed {
			next.ServeHTTP(w, r)
			return
		}

		if mode == "dry_run" {
			log.Printf("rate limit dry-run hit: path=%s key=%s", r.URL.Path, key)
			next.ServeHTTP(w, r)
			return
		}

		writeJSON(w, http.StatusTooManyRequests, &responsePayload{
			ErrCode:      4290,
			ResponseData: "rate limit exceeded",
		})
	})
}
