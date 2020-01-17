package libs

import (
	"net"
	"errors"
	"fmt"
)

func GetLocalIp() (string, error) {
	addrs, err := net.InterfaceAddrs()
    if err != nil {
		fmt.Println(err.Error())
		return "", err
    }

    for _, addr := range addrs {
        if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
            if ipnet.IP.To4() != nil {
                return ipnet.IP.String(), nil;
            }
        }
	}
	return "", errors.New("get local ip error!")
}

func MapInterfaceToMapString(mapInterface map[string]interface{})  map[string]string {

    mapString := make(map[string]string)

    for key, value := range mapInterface {
        strKey := fmt.Sprintf("%v", key)
        strValue := fmt.Sprintf("%v", value)

        mapString[strKey] = strValue
    }

    return mapString
}