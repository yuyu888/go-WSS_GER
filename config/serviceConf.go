package config

import (
	"github.com/spf13/viper"
	"log"
)

var (
	ServiceConf *ServiceConfig
)

type ServiceConfig struct {
	EtcdConf  EtcdConfig  `mapstructure:"etcd"`
	RedisConf RedisConfig `mapstructure:"redis"`
	//TransitConf TransitConfig `mapstructure:"transit"`
	RpcConf  RpcConfig  `mapstructure:"rpc"`
	HttpConf HttpConfig `mapstructure:"http"`
	LocalIp  string
	BaseConf BaseConfig `mapstructure:"base"`
}

type EtcdConfig struct {
	ServerAddr []string `mapstructure:"server"`
	Port       string   `mapstructure:"port"`
}

type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

//	type TransitConfig struct {
//	   Addr string `mapstructure:"addr"`
//	}
type RpcConfig struct {
	Addr         string `mapstructure:"addr"`
	Port         string `mapstructure:"port"`
	NetWork      string `mapstructure:"network"`
	RegisterName string `mapstructure:"registername"`
	BasePath     string `mapstructure:"basepath"`
}

type BaseConfig struct {
	Env    string `mapstructure:"env"`
	LogDir string `mapstructure:"logdir"`
}

type HttpConfig struct {
	Addr            string          `mapstructure:"addr"`
	ReadTimeoutSec  int             `mapstructure:"read_timeout_sec"`
	WriteTimeoutSec int             `mapstructure:"write_timeout_sec"`
	IdleTimeoutSec  int             `mapstructure:"idle_timeout_sec"`
	MaxBodyBytes    int64           `mapstructure:"max_body_bytes"`
	RateLimit       RateLimitConfig `mapstructure:"ratelimit"`
}

type RateLimitConfig struct {
	Enabled       bool   `mapstructure:"enabled"`
	Mode          string `mapstructure:"mode"`
	Strategy      string `mapstructure:"strategy"`
	WindowSeconds int64  `mapstructure:"window_seconds"`
	MaxRequests   int64  `mapstructure:"max_requests"`
	KeyBy         string `mapstructure:"key_by"`
}

func InitServiceConfig(env string) *ServiceConfig {
	confPath := "./config/"
	ServiceConf = &ServiceConfig{
		BaseConf: BaseConfig{
			Env: env,
		},
	}
	viper.SetConfigName("config_" + env)
	viper.SetConfigType("toml")
	viper.AddConfigPath(confPath)
	if err := viper.ReadInConfig(); err != nil {
		log.Fatal("read config error:" + err.Error())
	}
	if err := viper.Unmarshal(&ServiceConf); err != nil {
		log.Fatal("parse config error:" + err.Error())
	}
	ServiceConf.BaseConf.Env = env
	if ServiceConf.HttpConf.Addr == "" {
		ServiceConf.HttpConf.Addr = "0.0.0.0:80"
	}
	if ServiceConf.HttpConf.ReadTimeoutSec <= 0 {
		ServiceConf.HttpConf.ReadTimeoutSec = 5
	}
	if ServiceConf.HttpConf.WriteTimeoutSec <= 0 {
		ServiceConf.HttpConf.WriteTimeoutSec = 30
	}
	if ServiceConf.HttpConf.IdleTimeoutSec <= 0 {
		ServiceConf.HttpConf.IdleTimeoutSec = 60
	}
	if ServiceConf.HttpConf.MaxBodyBytes <= 0 {
		ServiceConf.HttpConf.MaxBodyBytes = 1 << 20
	}
	if ServiceConf.HttpConf.RateLimit.Mode == "" {
		ServiceConf.HttpConf.RateLimit.Mode = "enforce"
	}
	if ServiceConf.HttpConf.RateLimit.Strategy == "" {
		ServiceConf.HttpConf.RateLimit.Strategy = "fixed_window"
	}
	if ServiceConf.HttpConf.RateLimit.WindowSeconds <= 0 {
		ServiceConf.HttpConf.RateLimit.WindowSeconds = 1
	}
	if ServiceConf.HttpConf.RateLimit.MaxRequests <= 0 {
		ServiceConf.HttpConf.RateLimit.MaxRequests = 30
	}
	if ServiceConf.HttpConf.RateLimit.KeyBy == "" {
		ServiceConf.HttpConf.RateLimit.KeyBy = "ip"
	}
	//ServiceConf.RpcConf.Addr = localIp + ":" + ServiceConf.RpcConf.Port;
	log.Printf("config %v\n", ServiceConf)
	return ServiceConf
}
