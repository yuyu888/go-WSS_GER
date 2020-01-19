package wsServer

import(
	"sync"
)

//
type ClientPools struct {
    scene sync.Map
    len int
}

var WsClientPools = ClientPools{
    len: 0,
}

func (WsClientPools *ClientPools)save(uuid string, client *Client) {
    WsClientPools.scene.Store(uuid, client);
    WsClientPools.len = WsClientPools.len + 1;
}

func (WsClientPools *ClientPools)remove(uuid string){
    WsClientPools.scene.Delete(uuid);
    WsClientPools.len = WsClientPools.len - 1;
}

func (WsClientPools *ClientPools)get(uuid string) (*Client, bool) {
    var cl  *Client
	client, ok := WsClientPools.scene.Load(uuid);
	if (ok){
		return client.(*Client), ok;
	}
	return cl, ok;
}