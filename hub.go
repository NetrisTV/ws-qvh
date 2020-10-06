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
	wda = NewWdaHub(udid)
	h.webDriverAgents[udid] = wda
	go func() {
		<-wda.exitSignal
		h.deleteWdAgent(wda)
	}()
	return wda
}

func (h *Hub) unregisterClient(client *Client) {
	if _, ok := h.clients[client]; ok {
		receiver := client.receiver
		if receiver != nil {
			receiver.DelClient(client)
		}
		wda := client.wda
		if wda != nil {
			wda.DelClient(client)
		}
		client.stop()
		delete(h.clients, client)
		log.Info("Unregister client. Left: ", len(h.clients))
	}
}

func (h *Hub) deleteReceiver(receiver *ReceiverHub) {
	udid := receiver.udid
	delete(h.receivers, udid)
	wda := h.webDriverAgents[udid]
	if wda != nil {
		h.deleteWdAgent(wda)
	}
}

func (h *Hub) deleteWdAgent(wda *WdaHub) {
	udid := wda.udid
	delete(h.webDriverAgents, udid)
}

func (h *Hub) run(stopSignal chan interface{}) {
	h.stopSignal = stopSignal
	for {
		select {
		case <-stopSignal:
			log.Debug("Hub <- stopSignal")
			for client := range h.clients {
				h.unregisterClient(client)
			}
			for _, receiver := range h.receivers {
				select {
				case receiver.stopSignal <- nil:
					break
				default:
					break
				}
			}
			// all related WDA will stop because of usb reconfiguration

			select {
			case stopSignal <- nil:
				break
			default:
				break
			}
		case client := <-h.register:
			h.clients[client] = true
			log.Debug("Hub <- <-h.register. Total: ", len(h.clients))
		case client := <-h.unregister:
			log.Debug("Hub <- <-h.unregister. Total: ", len(h.clients))
			h.unregisterClient(client)
		case message := <-h.broadcast:
			for client := range h.clients {
				if client.send == nil {
					continue
				}
				client.mutex.Lock()
				select {
				case *client.send <- message:
				default:
					client.stop()
					delete(h.clients, client)
				}
				client.mutex.Unlock()
			}
		}
		log.Debug("Clients count: ", len(h.clients))
	}
}
