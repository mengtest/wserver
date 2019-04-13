package main

import (
	"fmt"
	"log"
	"net/http"
)

type HttpServer struct {
	Server   *http.Server
	Shutdown chan int
}

func (this *HttpServer) run() {
	defer close(this.Shutdown)
	err := this.Server.ListenAndServe()
	if err != nil {
		log.Printf("ListenAndServe: %v", err)
	}
}

func (this *HttpServer) close() {
	log.Printf("===============start to shutdown http server================")
	this.Server.Close()
	<-this.Shutdown
	log.Printf("==================shutdown http server success==============\n")
}

func newHttpServer(hub *Hub, nodeConfig *NodeConfig) *HttpServer {
	addr := fmt.Sprintf("0.0.0.0:%d", nodeConfig.WsPort)
	log.Printf("start http server bind to %s", addr)

	m := http.NewServeMux()
	server := &http.Server{
		Addr:    addr,
		Handler: m,
	}
	m.HandleFunc("/", serveHome)
	m.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	return &HttpServer{
		Server:   server,
		Shutdown: make(chan int),
	}
}
