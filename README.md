支持Websocket服务器多实例的
1. 集群动态发现
2. 集群广播

以实现高可用,高兵法支持

# 配置
yaml格式
``` yaml
nodes:
  - name: node1 # 名称,集群内不能重复
    domain: 1.l.self.kim  # 域名, 可以配置/etc/hosts,或者直接写死IP地址
    ws_port: 50001 # 给websocket的端口, 在docker/k8s上可以用同一个
    port: 51001 # 集群发现使用的端口,不应该暴露到外网
    sequence: 1 # 集群的顺序,暂未用到
  - name: node2
    domain: 2.l.self.kim
    ws_port: 50002
    port: 51002
    sequence: 2
```

## 编译
``` bash
go get github.com/golang/protobuf/protoc-gen-go
go get github.com/golang/protobuf/proto
dep ensure
go generate .
go build .
```

### 配置梯子
``` bash
export http_proxy="127.0.0.1:1080"
git config --global http.proxy "127.0.0.1:1080"
```

## 运行
``` bash
./wserver
./wserver --node=node2
./wserver --node=node3 --config=c.yaml
```

# Docker
