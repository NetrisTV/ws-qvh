package main

import (
	log "github.com/sirupsen/logrus"
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

}

func newHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    make(map[*Client]bool),
	}
}

func (h *Hub) run(stopSignal chan interface{}) {
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



