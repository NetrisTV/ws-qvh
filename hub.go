package main

import (
	log "github.com/sirupsen/logrus"
)

// Hub maintains the set of active clients and broadcasts messages to the
// clients.
type Hub struct {
	stopSignal chan interface{}

	receivers map[string]*ReceiverHub

	wdagents map[string]*WdaHub

	// Registered clients.
	clients map[*Client]bool

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
		receivers:  make(map[string]*ReceiverHub),
		wdagents:   make(map[string]*WdaHub),
	}
}

func (h *Hub) getOrCreateReceiver(udid string) *ReceiverHub {
	var receiver *ReceiverHub
	receiver = h.receivers[udid]
	if receiver != nil {
		return receiver
	}
	receiver = NewReceiver(h.stopSignal, udid)
	h.receivers[udid] = receiver
	return receiver
}

func (h *Hub) getOrCreateWdAgent(udid string) *WdaHub {
	var wda *WdaHub
	wda = h.wdagents[udid]
	if wda != nil {
		return wda
	}
	wda = NewWdaHub(h.stopSignal, udid)
	h.wdagents[udid] = wda
	return wda
}

func (h *Hub) run(stopSignal chan interface{}) {
	h.stopSignal = stopSignal
	for {
		select {
		case <-stopSignal:
			log.Info("Hub <- stopSignal")
			for client := range h.clients {
				client.stop()
				delete(h.clients, client)
			}
		case client := <-h.register:
			h.clients[client] = true
			log.Info("New client. ", len(h.clients))
		case client := <-h.unregister:
			log.Info("Hub <- h.unregister")
			if _, ok := h.clients[client]; ok {
				receiver := client.receiver
				if receiver != nil {
					receiver.DelClient(client)
					if len(receiver.clients) == 0 {
						udid := receiver.udid
						delete(h.receivers, udid)
					}
				}
				wda := client.wda
				if wda != nil {
					wda.DelClient(client)
					if len(wda.clients) == 0 {
						udid := receiver.udid
						delete(h.wdagents, udid)
					}
				}
				client.stop()
				delete(h.clients, client)
				log.Info("Client left. ", len(h.clients))
			}
		case message := <-h.broadcast:
			for client := range h.clients {
				select {
				case client.send <- message:
				default:
					client.stop()
					delete(h.clients, client)
				}
			}
		}
		if len(h.clients) == 0 {
			log.Info("Last client has left.")
		} else {
			log.Info("Clients count: ", len(h.clients))
		}
	}
}



