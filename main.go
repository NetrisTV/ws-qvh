package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/danielpaulus/go-ios/ios"
	"github.com/danielpaulus/quicktime_video_hack/screencapture"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
	"strings"
)

type detailsEntry struct {
	Udid           string
	ProductName    string
	ProductType    string
	ProductVersion string
}

func main() {
	log.SetLevel(log.DebugLevel)
	addr := "127.0.0.1:8080"
	if len(os.Args) > 1 {
		addr = os.Args[1]
	}
	startWebSocketServer(addr)
}

func startWebSocketServer(addr string) {
	log.Println("Starting WebSocket server")
	stopSignal := make(chan interface{})
	stopHub := make(chan interface{})
	shutdown := make(chan interface{})
	waitForSigInt(stopSignal)
	hub := newHub()
	go hub.run(stopHub)

	m := http.NewServeMux()
	s := http.Server{Addr: addr, Handler: m}

	m.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	go func() {
		err := s.ListenAndServe()
		if err != nil {
			log.Info("s.ListenAndServe(): ", err)
		}
		stopHub <- nil
		<-stopHub
		shutdown <- nil
	}()

	<-stopSignal
	err := s.Shutdown(context.TODO())
	if err != nil {
		log.Error(err)
	} else {
		log.Info("No error on shutdown")
	}
	<-shutdown
	log.Info("Program finished")
}

func getValues(device ios.DeviceEntry) ios.GetAllValuesResponse {

	deviceConn, err := ios.NewDeviceConnection(ios.DefaultUsbmuxdSocket)
	defer deviceConn.Close()
	if err != nil {
		log.Errorf("could not connect to %s with err %+v", ios.DefaultUsbmuxdSocket, err)
	}
	muxConnection := ios.NewUsbMuxConnection(deviceConn)
	defer muxConnection.Close()

	pairRecord, err := muxConnection.ReadPair(device.Properties.SerialNumber)
	if err != nil {
		log.Errorf("could not read pair records: %+v", err)
	}
	lockdownConnection, err := muxConnection.ConnectLockdown(device.DeviceID)
	if err != nil {
		log.Fatal(err)
	}
	lockdownConnection.StartSession(pairRecord)

	allValues, err := lockdownConnection.GetValues()
	if err != nil {
		log.Errorf("could not get all values from lock down connection: %+v", err)
	}
	lockdownConnection.StopSession()
	return allValues
}

func screenCaptureDevices() []byte {
	deviceList, err := screencapture.FindIosDevices()
	if err != nil {
		log.Fatalf("Error finding iOS Devices, error: %s", err)
	}

	result := make([]detailsEntry, len(deviceList))
	for i, device := range deviceList {
		udid := strings.Trim(device.SerialNumber, "\x00")
		if len(udid) == 24 {
			udid = fmt.Sprintf("%s-%s", udid[0:8], udid[8:])
		}
		result[i] = detailsEntry{
			Udid:           udid,
			ProductName:    device.ProductName,
			ProductType:    "",
			ProductVersion: "",
		}
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

func formatUdid(udid string) (string, error) {
	if len(udid) == 40 {
		return udid, nil
	}
	if len(udid) == 25 {
		return strings.Replace(udid, "-", "", 1), nil
	}
	return udid, fmt.Errorf("Invalid udid: %s", udid)
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

func toJSON(output interface{}) []byte {
	text, err := json.Marshal(output)
	if err != nil {
		log.Fatalf("Broken json serialization, error: %s", err)
	}
	return text
}
