package wsServer

import (
	"sync"
)

type ClientPools struct {
	scene sync.Map
}

var WsClientPools = ClientPools{}

func (WsClientPools *ClientPools) save(uuid string, client *Client) {
	WsClientPools.scene.Store(uuid, client)
}

func (WsClientPools *ClientPools) remove(uuid string) {
	WsClientPools.scene.Delete(uuid)
}

func (WsClientPools *ClientPools) get(uuid string) (*Client, bool) {
	var cl *Client
	client, ok := WsClientPools.scene.Load(uuid)
	if ok {
		return client.(*Client), ok
	}
	return cl, ok
}
