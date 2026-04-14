package wsServer

import "wssgo/config"

// UrlList 保存允许代理的 URL 白名单（scheme+host+path 精确匹配）
var UrlList map[string]bool

// originList 保存允许连接的 Origin 白名单，为 nil 表示允许所有
var originList map[string]bool

// InitUrlList 从配置文件初始化 URL 白名单和 Origin 白名单，在 wsServer.Init() 中调用
func InitUrlList() {
	UrlList = make(map[string]bool)
	for _, u := range config.ServiceConf.WsConf.AllowedUrls {
		UrlList[u] = true
	}

	if len(config.ServiceConf.WsConf.AllowedOrigins) > 0 {
		originList = make(map[string]bool)
		for _, o := range config.ServiceConf.WsConf.AllowedOrigins {
			originList[o] = true
		}
	}
}

// isOriginAllowed 检查 Origin 是否在白名单中，白名单为空时允许所有
func isOriginAllowed(origin string) bool {
	if originList == nil {
		return true
	}
	return originList[origin]
}
