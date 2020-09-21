package main

import (
	"github.com/danielpaulus/quicktime_video_hack/screencapture"
	log "github.com/sirupsen/logrus"
)

type ReceiverHub struct {
	udid string
	streaming bool
	closed bool
	send chan []byte
	clients map[*Client]bool
	stopSignal chan interface{}
}

func NewReceiver(stopSignal chan interface{}, udid string) *ReceiverHub {
	return &ReceiverHub{
		clients:    make(map[*Client]bool),
		send:       make(chan []byte),
		stopSignal: stopSignal,
		udid:       udid,
	}
}

func (r *ReceiverHub) AddClient(c *Client) {
	r.clients[c] = true
	if !r.streaming {
		r.stream()
	}
}

func (r *ReceiverHub) DelClient(c *Client) {
	delete(r.clients, c)
	if len(r.clients) == 0 {
		r.streaming = false
	}
}

func (r *ReceiverHub) stream() {
	if r.streaming {
		return
	}
	var udid = r.udid
	log.Info("Client stream ", udid)
	device, err := screencapture.FindIosDevice(udid)
	if err != nil {
		r.send <-  toErrJSON(err, "no device found to activate")
	}

	log.Debugf("Enabling device: %v", device)
	device, err = screencapture.EnableQTConfig(device)
	if err != nil {
		r.send <-  toErrJSON(err, "Error enabling QT config")
	}

	r.streaming = true
	writer := NewNaluHubWriter(r)
	adapter := screencapture.UsbAdapter{}
	stopSignal := r.stopSignal
	mp := screencapture.NewMessageProcessor(&adapter, stopSignal, writer, false)
	go func() {
		adapter.StartReading(device, &mp, stopSignal)
		<- stopSignal
	}()
}

func (r ReceiverHub) run() {
	for {
		select {
		case <- r.stopSignal:
			for client := range r.clients {
				delete(r.clients, client)
			}
		case data := <- r.send:
			for client := range r.clients {
				client.send <- data
			}
		}
	}
}
