// Copyright 2013 The Gorilla WebSocket Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"bytes"
	"encoding/json"
	"github.com/danielpaulus/quicktime_video_hack/screencapture"
	"github.com/gorilla/websocket"
	log "github.com/sirupsen/logrus"
	"net/http"
	"net/url"
	"time"
)

const (
	// Time allowed to write a message to the peer.
	writeWait = 100 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 600 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// Client is a middleman between the websocket connection and the client.
type Client struct {
	hub *Hub

	// The websocket connection.
	conn *websocket.Conn

	// Buffered channel of outbound messages.
	send chan []byte

	stopSignal chan interface{}

	closed bool

	mp *screencapture.MessageProcessor
}

// readPump pumps messages from the websocket connection to the hub.
//
// The application runs readPump in a per-connection goroutine. The application
// ensures that there is at most one reader on a connection by executing all
// reads from this goroutine.
func (c *Client) readPump() {
	defer func() {
		log.Info("readPump. defer")
		c.hub.unregister <- c
		c.stop()
		c.conn.Close()
	}()
	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error { c.conn.SetReadDeadline(time.Now().Add(pongWait)); return nil })
	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("error: %v", err)
			}
			break
		}
		var m Message
		err = json.Unmarshal(message, &m)
		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		if err != nil {
			log.Printf("error: %v", err)
			c.hub.broadcast <- message
		} else {
			switch m.Command {
			case "list":
				c.send <- devices()
			case "activate":
				c.send <- activate(m.UDID)
			case "stream":
				log.Info("Start")
				c.stream(m.UDID)
			case "run-wda":
				log.Info("Run wda")
				c.runWda(m.UDID)
			default:
				c.hub.broadcast <- message
			}
		}
	}
}

// writePump pumps messages from the hub to the websocket connection.
//
// A goroutine running writePump is started for each connection. The
// application ensures that there is at most one writer to a connection by
// executing all writes from this goroutine.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		log.Info("writePump. defer")
		ticker.Stop()
		c.conn.Close()
	}()
	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The client closed the channel.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.BinaryMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}
		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *Client) stop() {
	c.closed = true
	if c.stopSignal != nil {
		c.stopSignal <- nil
	}
}

func (c *Client) runWda(udid string) {

	var out Writer
	out.str = []rune(Begin)
	out.pos = 0
	out.value = ""
	ch := make(chan []byte)
	go func() {
		result := <- ch
		if result != nil {
			c.send <- result
		}
	}()
	go func() {
		out.Start(udid, ch)
	}()

}

func (c *Client) stream(udid string) {
	log.Info("Client stream ", udid)
	device, err := screencapture.FindIosDevice(udid)
	if err != nil {
		c.send <-  toErrJSON(err, "no device found to activate")
	}

	log.Debugf("Enabling device: %v", device)
	device, err = screencapture.EnableQTConfig(device)
	if err != nil {
		c.send <-  toErrJSON(err, "Error enabling QT config")
	}

	writer := NewNaluHubWriter(c)
	adapter := screencapture.UsbAdapter{}
	stopSignal := make(chan interface{})
	mp := screencapture.NewMessageProcessor(&adapter, stopSignal, writer, false)
	c.stopSignal = stopSignal
	go func() {
		adapter.StartReading(device, &mp, stopSignal)
		<- stopSignal
	}()
}

// serveWs handles websocket requests from the peer.
func serveWs(hub *Hub, w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println(err)
		return
	}
	client := &Client{hub: hub, conn: conn, send: make(chan []byte, 256)}
	client.hub.register <- client

	// Allow collection of memory referenced by the caller by doing all work in
	// new goroutines.
	go client.writePump()
	go client.readPump()

	if r.URL.RawQuery != "" {
		m, err := url.ParseQuery(r.URL.RawQuery)
		if err != nil {
			log.Errorln("Failed to parse query string:" + r.URL.RawQuery)
			return
		}
		udid := m.Get("stream")
		if udid != "" {
			client.stream(udid)
		}
	}
}

