package libs

import (
	"github.com/go-redis/redis"
	"log"
	"sync"
	"time"
	"wssgo/config"
)

type RedisConf struct {
	Mode               string   // standalone 或 sentinel
	Addr               string   // 单机地址
	SentinelAddrs      []string // 哨兵节点地址列表
	MasterName         string   // 哨兵主节点名称
	Password           string
	Db                 int
	DialTimeout        time.Duration
	ReadTimeout        time.Duration
	WriteTimeout       time.Duration
	PoolTimeout        time.Duration
	IdleCheckFrequency time.Duration
	PoolSize           int
	MinIdleConns       int
	MaxRetries         int
}

type RedisObj struct {
	Conf *RedisConf
	Cli  *redis.Client
}

// 全局单例
var (
	defaultRedis     *RedisObj
	defaultRedisOnce sync.Once
)

// GetDefaultRedisConf 从配置文件读取并返回 Redis 配置
func GetDefaultRedisConf() *RedisConf {
	c := config.ServiceConf.RedisConf
	return &RedisConf{
		Mode:               c.Mode,
		Addr:               c.Addr,
		SentinelAddrs:      c.SentinelAddrs,
		MasterName:         c.MasterName,
		Password:           c.Password,
		Db:                 c.DB,
		DialTimeout:        10 * time.Second,
		ReadTimeout:        30 * time.Second,
		WriteTimeout:       30 * time.Second,
		PoolSize:           100,
		PoolTimeout:        30 * time.Second,
		MinIdleConns:       10,
		IdleCheckFrequency: 40 * time.Second,
		MaxRetries:         3,
	}
}

// DefaultRedis 返回全局共享的 Redis 实例（单例）
func DefaultRedis() *RedisObj {
	defaultRedisOnce.Do(func() {
		r := NewRedis(GetDefaultRedisConf())
		r.Connect()
		defaultRedis = r
	})
	return defaultRedis
}

func NewRedis(config *RedisConf) *RedisObj {
	return &RedisObj{Conf: config}
}

// 重新定义 redis 设置
func (r *RedisObj) SetDialTimeOut(DialTimeout int) *RedisObj {
	r.Conf.DialTimeout = time.Duration(DialTimeout) * time.Second
	return r
}

// Connect 根据配置的 mode 选择单机或哨兵模式连接
func (r *RedisObj) Connect() {
	switch r.Conf.Mode {
	case "sentinel":
		if r.Conf.MasterName == "" {
			log.Fatal("redis sentinel mode requires master_name")
		}
		if len(r.Conf.SentinelAddrs) == 0 {
			log.Fatal("redis sentinel mode requires sentinel_addrs")
		}
		r.Cli = redis.NewFailoverClient(&redis.FailoverOptions{
			MasterName:         r.Conf.MasterName,
			SentinelAddrs:      r.Conf.SentinelAddrs,
			Password:           r.Conf.Password,
			DB:                 r.Conf.Db,
			DialTimeout:        r.Conf.DialTimeout,
			ReadTimeout:        r.Conf.ReadTimeout,
			WriteTimeout:       r.Conf.WriteTimeout,
			PoolSize:           r.Conf.PoolSize,
			PoolTimeout:        r.Conf.PoolTimeout,
			MinIdleConns:       r.Conf.MinIdleConns,
			IdleCheckFrequency: r.Conf.IdleCheckFrequency,
			MaxRetries:         r.Conf.MaxRetries,
		})
	default:
		// standalone 或未配置，默认单机
		r.Cli = redis.NewClient(&redis.Options{
			Addr:               r.Conf.Addr,
			Password:           r.Conf.Password,
			DB:                 r.Conf.Db,
			DialTimeout:        r.Conf.DialTimeout,
			ReadTimeout:        r.Conf.ReadTimeout,
			WriteTimeout:       r.Conf.WriteTimeout,
			PoolSize:           r.Conf.PoolSize,
			PoolTimeout:        r.Conf.PoolTimeout,
			MinIdleConns:       r.Conf.MinIdleConns,
			IdleCheckFrequency: r.Conf.IdleCheckFrequency,
			MaxRetries:         r.Conf.MaxRetries,
		})
	}
}

func (r *RedisObj) Ping() bool {
	if r.Cli == nil {
		return false
	}
	if _, err := r.Cli.Ping().Result(); err != nil {
		return false
	}
	return true
}

// 哈希存
func (r RedisObj) HSet(key, field string, value interface{}) error {
	return r.Cli.HSet(key, field, value).Err()
}

// 哈希获取
func (r RedisObj) HGet(key, field string) (string, error) {
	return r.Cli.HGet(key, field).Result()
}

// 哈希获取所有
func (r RedisObj) HGetAll(key string) (map[string]string, error) {
	return r.Cli.HGetAll(key).Result()
}

// 哈希删除
func (r RedisObj) HDel(key, field string) error {
	return r.Cli.HDel(key, field).Err()
}

// 删除 redis key
func (r RedisObj) Del(key string) error {
	return r.Cli.Del(key).Err()
}

func (r RedisObj) Get(key string) (string, error) {
	return r.Cli.Get(key).Result()
}

func (r RedisObj) Set(key string, value string, expire int) error {
	return r.Cli.Set(key, value, time.Duration(expire)*time.Second).Err()
}

// 设置过期时间
func (r RedisObj) Expire(key string, expire time.Duration) error {
	return r.Cli.Expire(key, expire).Err()
}
