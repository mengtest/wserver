package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
)

type server struct {
	Mng *GrpcNodeManager
}

func (this *server) Ping(
	ctx context.Context,
	in *PingRequest,
) (*PongRequest, error) {
	return &PongRequest{
		Name: in.Name,
	}, nil
}

func (this *server) Broadcast(
	ctx context.Context,
	in *BroadcastRequest,
) (*OkRequest, error) {
	// log.Printf("recv boradcast message %v", in.Content)
	this.Mng.Broadcaster.BroadcastData(in.Content)
	return &OkRequest{}, nil
}

//------------------------------------------------------------------------------

const NODE_LIFE int32 = 5

type GrpcNode struct {
	Mng         *GrpcNodeManager
	Config      *NodeConfig
	MessageChan chan interface{}
	Life        int32
}

func (this *GrpcNode) ProcessHeartbeat(c WServerClient) error {
	if this.Life > -1 {
		this.Life -= 1
	}
	if this.Life == 0 {
		log.Printf("node [%s] heartbeat dead", this.Config.Name)
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	_, err := c.Ping(ctx, &PingRequest{
		Name: this.Config.Name,
	})
	if err != nil {
		// log.Printf("node [%s] could not greet: %v", this.Config.Name, err)
		return err
	}

	if this.Life <= 0 {
		log.Printf("node [%s] heartbeat alive agin", this.Config.Name)
	}
	this.Life = NODE_LIFE
	// log.Printf("node [%s] recv heartbeat confirm %s", this.Config.Name, r.Name)
	return nil
}

func (this *GrpcNode) SendMessage(v interface{}, c WServerClient) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	if message, ok := v.(*BroadcastRequest); ok {
		_, err := c.Broadcast(ctx, message)
		if err != nil {
			log.Printf("node [%s] could not greet: %v", this.Config.Name, err)
			return err
		}
	} else {
		log.Printf("node [%s] recv invalid message")
	}
	return nil
}

func (this *GrpcNode) RunOnce(addr string) (bool, error) {
	conn, err := grpc.Dial(addr, grpc.WithInsecure())
	if err != nil {
		log.Printf("node [%s] did not connect: %v", this.Config.Name, err)
		time.Sleep(time.Second)
		return false, err
	}
	defer conn.Close()
	c := NewWServerClient(conn)

	breakFlag := false
	for !breakFlag {
		select {
		case conn, ok := <-this.MessageChan:
			if !ok {
				log.Printf("node [%s] conn message chan fail, break", this.Config.Name)
				breakFlag = true
				return true, nil
			}
			if this.Life > 0 {
				this.SendMessage(conn, c)
			}
		case <-time.After(2 * time.Second):
			this.ProcessHeartbeat(c)
		}
	}

	return false, nil
}

func (this *GrpcNode) run(wg *sync.WaitGroup) {
	defer wg.Done()

	addr := fmt.Sprintf("%s:%d", this.Config.Domain, this.Config.Port)
	log.Printf("node [%s] starting, server %s", this.Config.Name, addr)
	for {
		fini, _ := this.RunOnce(addr)
		if fini {
			break
		}
	}

	log.Printf("node [%s] stopping", this.Config.Name)
}

func (this *GrpcNode) close() {
	close(this.MessageChan)
}

type GrpcNodeManager struct {
	CurrentNode *NodeConfig  // 当前配置
	Nodes       []*GrpcNode  // 所有的连接
	Listener    net.Listener // tcp监听对象
	Server      *grpc.Server
	Shutdown    chan int
	Broadcaster Broadcaster
}

func (this *GrpcNodeManager) BroadcastData(v []byte) error {
	for _, node := range this.Nodes {
		node.MessageChan <- &BroadcastRequest{
			Content: v,
		}
	}
	return nil
}

func (this GrpcNodeManager) close() {
	log.Printf("===============start to shutdown grpc server================")
	// 关闭tcp服务器
	this.Server.GracefulStop()
	<-this.Shutdown
	this.Listener.Close()
	log.Printf("==================shutdown grpc server success==============\n")
}

func (this *GrpcNodeManager) run() {
	wg := sync.WaitGroup{}
	// wg.Add(1)

	for _, v := range this.Nodes {
		wg.Add(1)
		go v.run(&wg)
	}

	// 主逻辑
	RegisterWServerServer(this.Server, &server{
		Mng: this,
	})
	if err := this.Server.Serve(this.Listener); err != nil {
		log.Printf("failed to serve: %v", err)
	}

	// 关闭每个node
	for _, v := range this.Nodes {
		v.close()
	}
	wg.Wait()

	// 通知close
	close(this.Shutdown)
	log.Printf("stop node manager run success")
}

func newGrpcNodeManager(nodeConfig *NodeConfig, config *Config) (*GrpcNodeManager, error) {
	// 发现集群TCP服务器
	addr := fmt.Sprintf("0.0.0.0:%d", nodeConfig.Port)
	log.Printf("node manager listen addr %s", addr)
	l, err := net.Listen("tcp", addr)
	if err != nil {
		log.Printf("node manager listen addr fail %s %v", addr, err)
		return nil, err
	}

	mng := &GrpcNodeManager{
		CurrentNode: nodeConfig,
		Listener:    l,
		Shutdown:    make(chan int),
		Server:      grpc.NewServer(),
	}

	// 初始化Node配置
	for _, v := range config.Nodes {
		if v.Name == mng.CurrentNode.Name {
			continue
		}

		node := &GrpcNode{
			Mng:         mng,
			Config:      v,
			MessageChan: make(chan interface{}),
			Life:        0,
		}
		mng.Nodes = append(mng.Nodes, node)
	}

	return mng, nil
}
