package main

import (
	log "github.com/sirupsen/logrus"
)

type Hub struct {
	stopSignal      chan interface{}
	receivers       map[string]*ReceiverHub
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

func (h *Hub) unregisterClient(client *Client) {
	log.Info("Unregister client.")
	if _, ok := h.clients[client]; ok {
		receiver := client.receiver
		if receiver != nil {
			receiver.DelClient(client)
		}
		client.stop()
		delete(h.clients, client)
		log.Info("Unregister client. Left: ", len(h.clients))
	}
}

func (h *Hub) deleteReceiver(receiver *ReceiverHub) {
	udid := receiver.udid
	delete(h.receivers, udid)
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
			log.Debug("Hub. client := <-h.register. Total: ", len(h.clients))
		case client := <-h.unregister:
			h.unregisterClient(client)
			log.Debug("Hub. client := <-h.unregister. Total: ", len(h.clients))
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
