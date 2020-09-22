package main

import (
	log "github.com/sirupsen/logrus"
)

type WdaHub struct {
	clients    map[*Client]bool
	stopSignal chan interface{}
	udid string
	wdaUrl *[]byte
	tempChannel chan *[]byte
}

func NewWdaHub(stopSignal chan interface{}, udid string) *WdaHub {
	return &WdaHub{
		clients: make(map[*Client]bool),
		stopSignal: stopSignal,
		udid: udid,
	}
}

func (w *WdaHub) AddClient (c *Client) {
	_, ok := w.clients[c]
	if ok {
		log.Warn("Client already added")
		return
	}
	w.clients[c] = false
	if w.wdaUrl != nil {
		w.clients[c] = true
		c.send <- *w.wdaUrl
		return
	}
	if w.tempChannel != nil {
		return
	}
	if w.tempChannel == nil {
		w.tempChannel = make(chan *[]byte)
		wdaProcess := NewWdaProcess(&w.tempChannel)
		go func() {
			wdaProcess.Start(w.udid)
		}()
	}
	result := <- w.tempChannel
	for client, receivedUrl := range w.clients {
		if !receivedUrl && !client.closed {
			client.send <- *result
		}
	}
}


func (w *WdaHub) DelClient(c *Client) {
	delete(w.clients, c)
}