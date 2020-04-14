package libs

import (
    "github.com/go-redis/redis"
    "time"
    "wssgo/config"
)

type RedisConf struct{
    Addr         string
    Password     string
    Db           int
    DialTimeout        time.Duration
    ReadTimeout        time.Duration
    WriteTimeout       time.Duration
    PoolTimeout        time.Duration
    IdleCheckFrequency time.Duration
    PoolSize     int
    MinIdleConns int
}

type RedisObj struct {
    Conf *RedisConf
    Cli *redis.Client
}

func GetDefaultRedisConf() *RedisConf{
    conf := &RedisConf{
        Addr : config.ServiceConf.RedisConf.Addr,
        Password : config.ServiceConf.RedisConf.Password,
        Db	:	config.ServiceConf.RedisConf.DB,
        DialTimeout: 10 * time.Second,
        ReadTimeout: 30 * time.Second,
        WriteTimeout: 30 * time.Second,
        PoolSize : 100,
        PoolTimeout: 30 * time.Second,
        MinIdleConns : 10,
        IdleCheckFrequency : 40 * time.Second,
    }
    return conf
}

func  NewRedis(config *RedisConf) *RedisObj {
    r :=&RedisObj{}
    r.Conf = config
    return r
}

//重新定义redis设置
func (r *RedisObj) SetDialTimeOut(DialTimeout int) *RedisObj{
    r.Conf.DialTimeout = time.Duration(DialTimeout)*time.Second
    return r
}

func (r *RedisObj) Connect(){
    r.Cli = redis.NewClient(&redis.Options{
        Addr : r.Conf.Addr,
        Password : r.Conf.Password,
        DB	:	r.Conf.Db,
        DialTimeout: r.Conf.DialTimeout,
        ReadTimeout: r.Conf.ReadTimeout,
        WriteTimeout: r.Conf.WriteTimeout,
        PoolSize : r.Conf.PoolSize,
        PoolTimeout: r.Conf.PoolTimeout,
        MinIdleConns : r.Conf.MinIdleConns,
        IdleCheckFrequency : r.Conf.IdleCheckFrequency,
    })
}

func (r *RedisObj) Ping() bool {
    if r.Cli == nil {
        return false;
    }
    if _, err := r.Cli.Ping().Result(); err != nil {
        return false;
    }
    return true;
}

//哈希存
func (r RedisObj)HSet(key, field string, value interface{}) error {
    if ok := r.Ping(); !ok {
        r.Connect();
    }
    return r.Cli.HSet(key, field, value).Err();
}
//哈希获取
func (r RedisObj)HGet(key, field string)(string, error){
    if ok := r.Ping(); !ok {
        r.Connect();
    }
    return r.Cli.HGet(key, field).Result();
}
//哈希获取所有
func (r RedisObj)HGetAll(key string)(map[string]string, error){
    if ok := r.Ping(); !ok {
        r.Connect();
    }
    return r.Cli.HGetAll(key).Result();
}
//哈希删除
func (r RedisObj)HDel(key, field string) error {
    if ok := r.Ping(); !ok {
        r.Connect();
    }
    return r.Cli.HDel(key, field).Err();
}
//删除redis key
func (r RedisObj)Del(key string) error {
    if ok := r.Ping(); !ok {
        r.Connect();
    }
    return r.Cli.Del(key).Err();
}


func (r RedisObj)Get(key string) (string, error) {
    if ok := r.Ping(); !ok {
        r.Connect();
    }
    return r.Cli.Get(key).Result();
}

func (r RedisObj)Set(key string, value string, expire int) error {
    if ok := r.Ping(); !ok {
        r.Connect();
    }
    expireTime := time.Duration(expire)*time.Second
    return r.Cli.Set(key, value, expireTime).Err();
}
//设置过期时间
func (r RedisObj)Expire(key string, expire time.Duration) error {
    if ok := r.Ping(); !ok {
        r.Connect();
    }
    return r.Cli.Expire(key, expire).Err();
}