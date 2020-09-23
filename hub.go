package main

import (
	log "github.com/sirupsen/logrus"
)

type Hub struct {
	stopSignal      chan interface{}
	receivers       map[string]*ReceiverHub
	webDriverAgents map[string]*WdaHub
	clients         map[*Client]bool
	broadcast       chan []byte
	register        chan *Client
	unregister      chan *Client
}

func newHub() *Hub {
	return &Hub{
		broadcast:       make(chan []byte),
		register:        make(chan *Client),
		unregister:      make(chan *Client),
		clients:         make(map[*Client]bool),
		receivers:       make(map[string]*ReceiverHub),
		webDriverAgents: make(map[string]*WdaHub),
	}
}

func (h *Hub) getOrCreateReceiver(udid string) *ReceiverHub {
	var receiver *ReceiverHub
	receiver = h.receivers[udid]
	if receiver != nil {
		return receiver
	}
	receiver = NewReceiver(udid)
	h.receivers[udid] = receiver
	return receiver
}

func (h *Hub) getOrCreateWdAgent(udid string) *WdaHub {
	var wda *WdaHub
	wda = h.webDriverAgents[udid]
	if wda != nil {
		return wda
	}
	wda = NewWdaHub(h.stopSignal, udid)
	h.webDriverAgents[udid] = wda
	return wda
}

func (h *Hub) unregisterClient(client *Client) {
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
				udid := wda.udid
				delete(h.webDriverAgents, udid)
			}
		}
		client.stop()
		delete(h.clients, client)
		log.Info("Unregister client. Left: ", len(h.clients))
	}
}

func (h *Hub) run(stopSignal chan interface{}) {
	h.stopSignal = stopSignal
	for {
		select {
		case <-stopSignal:
			log.Info("Hub <- stopSignal")
			for client := range h.clients {
				h.unregisterClient(client)
			}
			stopSignal <- nil
		case client := <-h.register:
			h.clients[client] = true
			log.Info("New client. ", len(h.clients))
		case client := <-h.unregister:
			log.Info("Hub <- h.unregister")
			h.unregisterClient(client)
		case message := <-h.broadcast:
			for client := range h.clients {
				send := client.send
				if send == nil {
					continue
				}
				select {
				case *send <- message:
				default:
					client.stop()
					delete(h.clients, client)
				}
			}
		}
		log.Info("Clients count: ", len(h.clients))
	}
}
