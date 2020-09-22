package main

import (
	"github.com/danielpaulus/quicktime_video_hack/screencapture"
	log "github.com/sirupsen/logrus"
)

type ReceiverHub struct {
	udid       string
	streaming  bool
	closed     bool
	send       chan []byte
	clients    map[*Client]*ClientReceiveStatus
	stopStream chan interface{}
	stopSignal chan interface{}
	writer     *NaluWriter
	sei        []byte
	pps        []byte
	sps        []byte
}

type ClientReceiveStatus struct {
	gotPPS    bool
	gotSPS    bool
	gotSEI    bool
	gotIFrame bool
}

func NewReceiver(stopSignal chan interface{}, udid string) *ReceiverHub {
	return &ReceiverHub{
		clients:    make(map[*Client]*ClientReceiveStatus),
		send:       make(chan []byte),
		stopSignal: stopSignal,
		udid:       udid,
	}
}

func (r *ReceiverHub) storeNalUnit(dst *[]byte, b *[]byte) {
	*dst = make([]byte, len(*b))
	copy(*dst, *b)
}

func (r *ReceiverHub) AddClient(c *Client) {
	_, ok := r.clients[c]
	if ok {
		log.Warn("Client already added")
		return
	}
	status := &ClientReceiveStatus{}
	r.clients[c] = status
	if !r.streaming {
		r.streaming = true
		r.stopStream = make(chan interface{})
		go r.run()
		r.stream()
	}
}

func (r *ReceiverHub) DelClient(c *Client) {
	delete(r.clients, c)
	if len(r.clients) == 0 {
		r.streaming = false
		r.stopStream <- nil
	}
}

func (r *ReceiverHub) stream() {
	var udid = r.udid
	device, err := screencapture.FindIosDevice(udid)
	if err != nil {
		r.send <- toErrJSON(err, "no device found to activate")
	}

	log.Debugf("Enabling device: %v", device)
	device, err = screencapture.EnableQTConfig(device)
	if err != nil {
		r.send <- toErrJSON(err, "Error enabling QT config")
	}
	r.writer = NewNaluWriter(r)
	adapter := screencapture.UsbAdapter{}
	mp := screencapture.NewMessageProcessor(&adapter, r.stopStream, r.writer, false)
	go func() {
		adapter.StartReading(device, &mp, r.stopStream)
		//<- stopSignal
	}()
}

func (r *ReceiverHub) run() {
	for {
		select {
		case <-r.stopSignal:
			for client := range r.clients {
				delete(r.clients, client)
			}
		case data := <-r.send:
			for client, status := range r.clients {
				naluType := data[4] & 31
				if naluType == 8 {
					r.storeNalUnit(&r.pps, &data)
				} else if naluType == 7 {
					r.storeNalUnit(&r.sps, &data)
				} else if naluType == 6 {
					r.storeNalUnit(&r.sei, &data)
				}
				if status.gotIFrame {
					client.send <- data
				} else {
					if !status.gotPPS && r.pps != nil {
						status.gotPPS = true
						client.send <- r.pps
						if naluType == 8 {
							continue
						}
					}
					if !status.gotSPS && r.sps != nil {
						status.gotSPS = true
						client.send <- r.sps
						if naluType == 7 {
							continue
						}
					}
					if !status.gotSEI && r.sei != nil {
						status.gotSEI = true
						client.send <- r.sei
						if naluType == 6 {
							continue
						}
					}
					isIframe := naluType == 5
					if status.gotPPS && status.gotSPS && status.gotSEI && isIframe {
						status.gotIFrame = true
						client.send <- data
					} else {
						// log.Info("Receiver. ", "skipping frame for client: ", naluType)
					}
				}
			}
		}
	}
}
