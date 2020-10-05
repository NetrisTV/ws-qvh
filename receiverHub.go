package main

import (
	"github.com/danielpaulus/quicktime_video_hack/screencapture"
	log "github.com/sirupsen/logrus"
	"time"
)

const (
	PPS = 8
	SPS = 7
	SEI = 6
	IDR = 5
)

type ReceiverHub struct {
	udid           string
	streaming      bool
	closed         bool
	send           chan []byte
	clients        map[*Client]*ClientReceiveStatus
	stopReading    chan interface{}
	stopSignal     chan interface{}
	timeoutChannel chan bool
	writer         *NaluWriter
	sei            []byte
	pps            []byte
	sps            []byte
}

type ClientReceiveStatus struct {
	gotPPS    bool
	gotSPS    bool
	gotSEI    bool
	gotIFrame bool
}

func NewReceiver(udid string) *ReceiverHub {
	return &ReceiverHub{
		clients:        make(map[*Client]*ClientReceiveStatus),
		send:           make(chan []byte),
		stopSignal:     make(chan interface{}),
		timeoutChannel: make(chan bool),
		udid:           udid,
	}
}

func (r *ReceiverHub) storeNalUnit(dst *[]byte, b *[]byte) {
	*dst = make([]byte, len(*b))
	copy(*dst, *b)
}

func (r *ReceiverHub) AddClient(c *Client) {
	_, ok := r.clients[c]
	if ok {
		log.Warn("ReceiverHub. ", "Client already added")
		return
	}
	status := &ClientReceiveStatus{}
	r.clients[c] = status
	if !r.streaming {
		r.streaming = true
		r.stopReading = make(chan interface{})
		go r.run()
		r.stream()
	}
	select {
	case r.timeoutChannel <- false:
		break
	default:
		break
	}
}

func (r *ReceiverHub) DelClient(c *Client) {
	delete(r.clients, c)
	if len(r.clients) == 0 {
		go func() {
			time.Sleep(10 * time.Second)
			select {
			case r.timeoutChannel <- true:
				break
			default:
				break
			}
		}()
		go func() {
			doStop := <-r.timeoutChannel
			if doStop {
				c.hub.deleteReceiver(r)
				r.streaming = false
				r.closed = true
				r.stopSignal <- nil
			}
		}()
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
	mp := screencapture.NewMessageProcessor(&adapter, r.stopReading, r.writer, false)
	go func() {
		err := adapter.StartReading(device, &mp, r.stopReading)
		if err != nil {
			log.Error("adapter.StartReading(device, &mp, r.stopReading): ", err)
		}
		r.writer.Stop()
	}()
}

func (r *ReceiverHub) run() {
	for {
		select {
		case <-r.stopSignal:
			for client := range r.clients {
				delete(r.clients, client)
			}
			r.closed = true
			r.streaming = false
			r.stopReading <- nil
			select {
			case r.timeoutChannel <- true:
				break
			default:
				break
			}
		case data := <-r.send:
			for client, status := range r.clients {
				send := client.send
				if send == nil {
					continue
				}
				nalUnitType := data[4] & 31
				if nalUnitType == PPS {
					r.storeNalUnit(&r.pps, &data)
				} else if nalUnitType == SPS {
					r.storeNalUnit(&r.sps, &data)
				} else if nalUnitType == SEI {
					r.storeNalUnit(&r.sei, &data)
				}
				if status.gotIFrame {
					*send <- data
				} else {
					if !status.gotPPS && r.pps != nil {
						status.gotPPS = true
						*send <- r.pps
						if nalUnitType == PPS {
							continue
						}
					}
					if !status.gotSPS && r.sps != nil {
						status.gotSPS = true
						*send <- r.sps
						if nalUnitType == SPS {
							continue
						}
					}
					if !status.gotSEI && r.sei != nil {
						status.gotSEI = true
						*send <- r.sei
						if nalUnitType == SEI {
							continue
						}
					}
					isIframe := nalUnitType == IDR
					if status.gotPPS && status.gotSPS && status.gotSEI && isIframe {
						status.gotIFrame = true
						*send <- data
					} else {
						// log.Info("Receiver. ", "skipping frame for client: ", nalUnitType)
					}
				}
			}
		}
	}
}
