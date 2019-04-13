package main

import (
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
)

// go get github.com/golang/protobuf/protoc-gen-go
// go get github.com/golang/protobuf/proto
//go:generate protoc -I=. --go_out=plugins=grpc:.  message.proto

// export http_proxy="127.0.0.1:8123"
// git config --global http.proxy "127.0.0.1:8123"

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

var config_path = flag.String("config", "config.yaml", "配置文件路径")
var node_name = flag.String("node", "node1", "节点名称,必须和配置匹配")

func main() {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	flag.Parse()

	// 读取配置
	config := loadConfig(*config_path)
	var nodeConfig *NodeConfig = nil
	for _, v := range config.Nodes {
		if v.Name == *node_name {
			nodeConfig = v
		}
		log.Printf("new nodes %s %s %d %d\n", v.Name, v.Domain, v.Port, v.WsPort)

		// 解析DNS (一般集群在内网, domain是配置的hosts, 或者从k8s的内部dns服务器解析
		// addr, err := net.LookupIP(v.Domain)
		// if err != nil || len(addr) < 1 {
		// 	log.Println("Unknown host %s", v.Domain)
		// } else {
		// 	v.Ip = addr[0].String()
		// 	log.Printf("parse ip %s from domain %s", v.Ip, v.Domain)
		// }
	}
	if nodeConfig == nil {
		panic(errors.New("invalid node name"))
	}
	log.Printf("current node name %v", nodeConfig.Name)

	// 发现集群
	nodeManager, err := newGrpcNodeManager(nodeConfig, config)
	if err != nil {
		panic(err)
	}

	// websocket中转服务器
	hub := newHub(nodeManager)
	// http 服务器
	httpServer := newHttpServer(hub, nodeConfig)
	nodeManager.Broadcaster = hub
	go nodeManager.run()
	go hub.run()
	go httpServer.run()

	// 等待结束
	<-sigs
	log.Println("recv signal, try to stop app")
	// 必须在关闭hub前关闭httpserver
	httpServer.close()
	nodeManager.close()
	hub.close()
	log.Println("stop app success")
}
