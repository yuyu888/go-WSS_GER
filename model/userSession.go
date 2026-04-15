package model

import (
	"encoding/json"
	"errors"
	"time"
	"wssgo/libs"
)

const (
	cacheExpire = 60 * time.Second
	cachePrefix = "ws_go_"
	wssidPrefix = "ws_go_wssid_"
)

type UserSession struct {
	Uid      string
	DeviceId string
	Session
	redisCli *libs.RedisObj
}

type Session struct {
	WsServerAddr string `json:"ws_server_addr"`
	WssId        string `json:"wssid"`
}

func NewUserSession() *UserSession {
	u := &UserSession{}
	// 复用全局单例，不再每次新建连接池
	u.redisCli = libs.DefaultRedis()
	return u
}

func (u UserSession) GetInfo(uid string, deviceid string) (userInfo map[string]string, err error) {
	userInfo = make(map[string]string)
	if len(uid) > 0 && len(deviceid) > 0 {
		info, err := u.redisCli.HGet(cachePrefix+uid, deviceid)
		if err != nil {
			return userInfo, err
		}
		userInfo[deviceid] = info
		return userInfo, err
	}

	if len(uid) > 0 {
		userInfo, err = u.redisCli.HGetAll(cachePrefix + uid)
		return userInfo, err
	}
	if len(deviceid) > 0 {
		info, err := u.redisCli.HGet(cachePrefix+deviceid, deviceid)
		if err != nil {
			return userInfo, err
		}
		userInfo[deviceid] = info
		return userInfo, err
	}
	return userInfo, nil
}

func (u UserSession) GetWsServer(session string) ([]byte, []byte, error) {
	userSession := Session{}
	err := json.Unmarshal([]byte(session), &userSession)
	if err != nil {
		if libs.Logger != nil {
			libs.Logger.Errorf("session unmarshal error: %s", session)
		}
		return []byte(""), []byte(""), err
	}
	return []byte(userSession.WsServerAddr), []byte(userSession.WssId), nil
}

func (u UserSession) SaveInfo(uid string, deviceid string, SessionData *Session) error {
	session, err := json.Marshal(SessionData)
	if err != nil {
		return err
	}
	key := cachePrefix + uid
	if err := u.redisCli.HSet(key, deviceid, session); err != nil {
		return err
	}
	return u.redisCli.Expire(key, cacheExpire)
}

func (u UserSession) DelInfo(uid string, deviceid string) error {
	if len(uid) > 0 && len(deviceid) > 0 {
		return u.redisCli.HDel(cachePrefix+uid, deviceid)
	}
	if len(uid) > 0 {
		return u.redisCli.Del(cachePrefix + uid)
	}
	if len(deviceid) > 0 {
		return u.redisCli.Del(cachePrefix + deviceid)
	}
	return errors.New("params error:uid" + uid + "\tDeviceId:" + deviceid)
}

func (u UserSession) ExpireInfo(uid string) error {
	return u.redisCli.Expire(cachePrefix+uid, cacheExpire)
}

// SaveWssid 保存 wssid → ws_server_addr 反向映射，供跨节点 broadcast 路由使用
func (u UserSession) SaveWssid(wssid, serverAddr string) error {
	return u.redisCli.Set(wssidPrefix+wssid, serverAddr, int(cacheExpire.Seconds()))
}

// GetWssidServer 根据 wssid 查询所在节点的 IP 地址
func (u UserSession) GetWssidServer(wssid string) (string, error) {
	return u.redisCli.Get(wssidPrefix + wssid)
}

// DelWssid 删除 wssid 反向映射
func (u UserSession) DelWssid(wssid string) error {
	return u.redisCli.Del(wssidPrefix + wssid)
}

// ExpireWssid 刷新 wssid 反向映射的 TTL
func (u UserSession) ExpireWssid(wssid string) error {
	return u.redisCli.Expire(wssidPrefix+wssid, cacheExpire)
}
