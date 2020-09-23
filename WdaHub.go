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
		log.Debug("Send stored wdaUrl to client")
		w.clients[c] = true
		if c.send != nil {
			*c.send <- *w.wdaUrl
		}
		return
	}
	if w.tempChannel != nil {
		return
	}
	if w.tempChannel == nil {
		log.Debug("Run new WDA process")
		w.tempChannel = make(chan *[]byte)
		wdaProcess := NewWdaProcess(&w.tempChannel)
		go func() {
			wdaProcess.Start(w.udid)
		}()
	}
	w.wdaUrl = <- w.tempChannel
	for client, receivedUrl := range w.clients {
		send := client.send
		if send == nil {
			continue
		}
		if !receivedUrl {
			select {
			case *send <- *w.wdaUrl:
			default:
				log.Warn("Failed to send wdaUrl to client")
			}
		}
	}
}


func (w *WdaHub) DelClient(c *Client) {
	delete(w.clients, c)
}