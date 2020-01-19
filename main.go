package main

import (
    "wssgo/wsServer"
    "wssgo/httpServer"

)

func main() {
    go httpServer.Init();
    wsServer.Init();
}