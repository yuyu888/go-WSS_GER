package config

import (
    "github.com/spf13/viper"
    "log"
)
var (
    ServiceConf *ServiceConfig
)

type ServiceConfig struct {
    EtcdConf EtcdConfig	`mapstructure:"etcd"`
    RedisConf RedisConfig `mapstructure:"redis"`
    //TransitConf TransitConfig `mapstructure:"transit"`
    RpcConf RpcConfig `mapstructure:"rpc"`
    LocalIp string
    BaseConf BaseConfig `mapstructure:"base"`
}



type EtcdConfig struct {
    ServerAddr []string `mapstructure:"server"`
    Port string `mapstructure:"port"`
}

type RedisConfig struct {
    Addr string `mapstructure:"addr"`
    Password string `mapstructure:"password"`
    DB int `mapstructure:"db"`
}

//type TransitConfig struct {
//    Addr string `mapstructure:"addr"`
//}
//
type RpcConfig struct {
    Addr string `mapstructure:"addr"`
    Port string `mapstructure:"port"`
    NetWork string `mapstructure:"network"`
    RegisterName string `mapstructure:"registername"`
    BasePath string `mapstructure:"basepath"`
}

type BaseConfig struct {
    Env string `mapstructure:"env"`
    LogDir string `mapstructure:"logdir"`
}

func InitServiceConfig(env string) *ServiceConfig {
    confPath := "./config/"
    ServiceConf = &ServiceConfig{
        BaseConf : BaseConfig{
            Env : env,
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
    ServiceConf.BaseConf.Env = env;
    //ServiceConf.RpcConf.Addr = localIp + ":" + ServiceConf.RpcConf.Port;
    log.Printf("config %v\n", ServiceConf)
    return ServiceConf;
}