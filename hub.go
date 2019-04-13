// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"log"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client

	Stoping  chan int
	Shutdown chan int

	Broadcaster Broadcaster
}

func (this *Hub) close() {
	log.Printf("===============start to shutdown ws server================")
	// 先处理clients
	for k, _ := range this.clients {
		k.close()
	}
	close(this.Stoping)
	<-this.Shutdown
	log.Printf("==================shutdown ws server success==============\n")
}

func newHub(b Broadcaster) *Hub {
	return &Hub{
		broadcast:   make(chan []byte),
		register:    make(chan *Client),
		unregister:  make(chan *Client),
		clients:     make(map[*Client]bool),
		Stoping:     make(chan int),
		Shutdown:    make(chan int),
		Broadcaster: b,
	}
}

func (this *Hub) BroadcastData(v []byte) error {
	this.BroadcastToClients(v)
	return nil
}

func (this *Hub) BroadcastToClients(message []byte) {
	for client := range this.clients {
		select {
		case client.send <- message:
		default:
			close(client.Stoping)
			delete(this.clients, client)
		}
	}
}

func (this *Hub) run() {
	defer func() {
		close(this.register)
		close(this.unregister)
		close(this.broadcast)
		close(this.Shutdown)
	}()

	breakFlag := false
	for !breakFlag {
		select {
		case client := <-this.register:
			{
				this.clients[client] = true
			}
		case _, ok := <-this.Stoping:
			{
				if !ok {
					breakFlag = true
					break
				}
			}
		case client := <-this.unregister:
			if _, ok := this.clients[client]; ok {
				delete(this.clients, client)
				close(client.send)
			}
		case message := <-this.broadcast:
			// log.Printf("recv hub message %v, try to broadcast", message)
			this.Broadcaster.BroadcastData(message)
			this.BroadcastToClients(message)
		}
	}
}
