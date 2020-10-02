package main

import (
	log "github.com/sirupsen/logrus"
)

type WdaHub struct {
	clients     map[*Client]bool
	exitSignal  chan interface{}
	udid        string
	message     *MessageRunWda
	tempChannel chan *MessageRunWda
}

func NewWdaHub(udid string) *WdaHub {
	return &WdaHub{
		clients: make(map[*Client]bool),
		exitSignal: make(chan interface{}),
		udid: udid,
	}
}

func (w *WdaHub) AddClient (c *Client) {
	_, ok := w.clients[c]
	if ok {
		log.Warn("WdaHub. ", "Client already added")
		return
	}
	w.clients[c] = false
	if w.message != nil {
		log.Debug("Send stored message to client")
		w.clients[c] = true
		if c.send != nil {
			*c.send <- toJSON(w.message)
		}
		return
	}
	if w.tempChannel != nil {
		return
	}
	if w.tempChannel == nil {
		log.Debug("Run new WDA process")
		w.tempChannel = make(chan *MessageRunWda)
		wdaProcess := NewWdaProcess(w.udid, &w.tempChannel, &w.exitSignal)
		go func() {
			wdaProcess.Start()
		}()
	}
	w.message = <- w.tempChannel
	message := toJSON(w.message)
	for client, receivedUrl := range w.clients {
		send := client.send
		if send == nil {
			continue
		}
		if !receivedUrl {
			*send <- message
		}
	}
}


func (w *WdaHub) DelClient(c *Client) {
	delete(w.clients, c)
}