package main

import (
	"context"
	"encoding/json"
	"github.com/danielpaulus/go-ios/usbmux"
	"github.com/danielpaulus/quicktime_video_hack/screencapture"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
)

type detailsEntry struct {
	Udid           string
	ProductName    string
	ProductType    string
	ProductVersion string
}

func main() {
	log.SetLevel(log.DebugLevel)
	addr := ":8080"
	dir := "dist"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	if len(os.Args) > 2 {
		dir = os.Args[2]
	}
	startWebSocketServer(addr, dir)
}

func startWebSocketServer(addr string, dir string) {
	log.Println("Starting WebSocket server")
	stopSignal := make(chan interface{})
	stopHub := make(chan interface{})
	shutdown := make(chan interface{})
	waitForSigInt(stopSignal)
	hub := newHub()
	go hub.run(stopHub)

	m := http.NewServeMux()
	s := http.Server{Addr: addr, Handler: m}

	m.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(dir))))

	m.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	go func() {
		err := s.ListenAndServe()
		log.Info("s.ListenAndServe(): ", err)
		stopHub <- nil
		<- stopHub
		log.Warn("shutdown <- nil")
		shutdown <- nil
	}()

	<-stopSignal
	log.Warn("startWebSocketServer. ", "<-stopSignal")
	err := s.Shutdown(context.TODO())
	if err != nil {
		log.Error(err)
	} else {
		log.Info("No error on shutdown")
	}
	<- shutdown
	log.Info("Program finished")
}

func getValues(device usbmux.DeviceEntry) usbmux.GetAllValuesResponse {
	muxConnection := usbmux.NewUsbMuxConnection()
	defer muxConnection.Close()

	pairRecord := muxConnection.ReadPair(device.Properties.SerialNumber)

	lockdownConnection, err := muxConnection.ConnectLockdown(device.DeviceID)
	if err != nil {
		log.Fatal(err)
	}
	lockdownConnection.StartSession(pairRecord)

	allValues := lockdownConnection.GetValues()
	lockdownConnection.StopSession()
	return allValues
}

func devices() []byte {
	deviceList := usbmux.ListDevices()
	result := make([]detailsEntry, len(deviceList.DeviceList))
	for i, device := range deviceList.DeviceList {
		udid := device.Properties.SerialNumber
		allValues := getValues(device)
		result[i] = detailsEntry{udid, allValues.Value.ProductName, allValues.Value.ProductType, allValues.Value.ProductVersion}
	}
	text, err := json.Marshal(result)
	if err != nil {
		log.Fatalf("Broken json serialization, error: %s", err)
	}
	return text
}

// This command is for testing if we can enable the hidden Quicktime device config
func activate(udid string) []byte {
	device, err := screencapture.FindIosDevice(udid)
	if err != nil {
		return toErrJSON(err, "no device found to activate")
	}

	log.Debugf("Enabling device: %v", device)
	device, err = screencapture.EnableQTConfig(device)
	if err != nil {
		return toErrJSON(err, "Error enabling QT config")
	}

	return toJSON(map[string]interface{}{
		"device_activated": device.DetailsMap(),
	})
}

func waitForSigInt(stopSignalChannel chan interface{}) {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	go func() {
		for sig := range c {
			log.Debugf("Signal received: %s", sig)
			var stopSignal interface{}
			stopSignalChannel <- stopSignal
		}
	}()
}

func toErrJSON(err error, msg string) []byte {
	log.Debug(msg, err)
	return toJSON(map[string]interface{}{
		"original_error": err.Error(),
		"error_message":  msg,
	})
}

func toJSON(output map[string]interface{}) []byte {
	text, err := json.Marshal(output)
	if err != nil {
		log.Fatalf("Broken json serialization, error: %s", err)
	}
	return text
}
