package model

import (
    "wssgo/libs"
    "errors"
    "encoding/json"
    "time"
    //"fmt"
)

const (
    cacheExpire = 60 * time.Second
    cachePrefix = "ws_go_"
)


type UserSession struct {
    Uid string
    DeviceId string
    Session
    redisCli *libs.RedisObj
}

type Session struct {
    WsServerAddr string  `json:"ws_server_addr"`
    WssId string  `json:"wssid"`

}
func  NewUserSession() *UserSession {
    u :=&UserSession{}
    redisConf := libs.GetDefaultRedisConf()
    u.redisCli = libs.NewRedis(redisConf)
    u.redisCli.Connect()
    return u
}

func (u UserSession)GetInfo(uid string, deviceid string) (userInfo map[string]string, err error) {
    userInfo = make(map[string]string);
    if len(uid) > 0 && len(deviceid) > 0 {
        info, err := u.redisCli.HGet(cachePrefix + uid, deviceid);
        if err != nil {
            return userInfo, err;
        }
        userInfo[deviceid] = info;
        return userInfo, err;
    }

    if len(uid) > 0 {
        userInfo, err = u.redisCli.HGetAll(cachePrefix + uid);
        if err != nil {
        }
        return userInfo, err;
    }
    if len(deviceid) > 0 {
        info, err := u.redisCli.HGet(cachePrefix + deviceid, deviceid);
        if err != nil {
            return userInfo, err;
        }
        userInfo[deviceid] = info;
        return userInfo, err;
    }
    return userInfo, nil;
}

func (u UserSession)GetWsServer(session string) ([]byte, []byte, error) {
    userSession := Session{};
    err := json.Unmarshal([]byte(session), &userSession);
    if err != nil {
        libs.Logger.Errorf("session unmarshal error!", session);
        return []byte(""), []byte(""), err;
    }
    return []byte(userSession.WsServerAddr), []byte(userSession.WssId), nil;
}

func (u UserSession)SaveInfo(uid string, deviceid string, SessionData *Session) error {
    session, err := json.Marshal(SessionData)
    if err != nil {
        return err
    }
    key := cachePrefix + uid
    if err := u.redisCli.HSet(key, deviceid, session); err != nil {
        return err
    }
    return u.redisCli.Expire(key, cacheExpire);
}

func (u UserSession)DelInfo(uid string, deviceid string) error {
    if len(uid) > 0 && len(deviceid) > 0 {
        return u.redisCli.HDel(cachePrefix + uid, deviceid)
    }
    if len(uid) > 0 {
        return u.redisCli.Del(cachePrefix + uid);
    }
    if len(deviceid) > 0 {
        return u.redisCli.Del(cachePrefix + deviceid);
    }
    return errors.New("params error:uid" + uid + "\tDeviceId:" + deviceid);
}

func (u UserSession)ExpireInfo(uid string) error {
    return u.redisCli.Expire(cachePrefix + uid, cacheExpire);
}