# go-WSS_GER
一个go+etcd+rpcx 实现的websocket的服务

#### 安装
go mod init wssgo

go run main.go

### 测试数据
{"wssid":"16b3d4db-4586-4002-8cc8-d5fd0cc877f3","request_id":"d4f50517-0005-49f1-bd18-85ab24cfe701","request_data":{"http_method":"POST","request_url":"http:\/\/localhost\/test?id=11111","post_data":"msg=ddddd&ww=eee","headers":{"test":"www"}},"request_type":"req&resp","action":"user.showInfo"}

### 感言
当初做这个项目的时候，go基本零基础；也是借鉴了一些网上的架构思路觉得不错，就勇敢的挑战了一下；用现在的眼光看当时的实现， 或有种不忍直视的感觉，很多实现好幼幼，不过设计思想还是严格的实现了，心理也暗自骄傲，后期的大量项目未必有这个项目架构设计，挑战性也没这个大（干的时候也只有个方向，什么基础都没有，一点成竹在胸的感觉都没有，全都要探索);把这个项目做完，收获也是丰厚的，对于go语言的感觉豁然开朗，算是完成了入门，什么事情都是要多练，敢干，才能更好的领悟；
