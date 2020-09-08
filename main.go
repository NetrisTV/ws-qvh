package main

import (
	"bufio"
	"context"
	"encoding/json"
	"github.com/danielpaulus/quicktime_video_hack/screencapture"
	"github.com/danielpaulus/quicktime_video_hack/screencapture/coremedia"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"os/signal"
)

func main() {
	log.SetLevel(log.InfoLevel)
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
	waitForSigInt(stopSignal)
	hub := newHub()
	go hub.run(stopSignal)

	m := http.NewServeMux()
	s := http.Server{Addr: addr, Handler: m}

	m.Handle("/", http.StripPrefix("/", http.FileServer(http.Dir(dir))))

	m.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	go func() {
		log.Fatal(s.ListenAndServe())
	}()

	<-stopSignal
	s.Shutdown(context.Background())
}

// Just dump a list of what was discovered to the console
func devices() []byte {
	deviceList, err := screencapture.FindIosDevices()
	if err != nil {
		printErrJSON(err, "Error finding iOS Devices")
	}
	log.Debugf("Found (%d) iOS Devices with UsbMux Endpoint", len(deviceList))

	if err != nil {
		printErrJSON(err, "Error finding iOS Devices")
	}
	output := screencapture.PrintDeviceDetails(deviceList)

	text, err := json.Marshal(output)
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

func record(h264FilePath string, wavFilePath string, udid string) {
	log.Debugf("Writing video output to:'%s' and audio to: %s", h264FilePath, wavFilePath)

	h264File, err := os.Create(h264FilePath)
	if err != nil {
		log.Debugf("Error creating h264File:%s", err)
		log.Errorf("Could not open h264File '%s'", h264FilePath)
	}
	wavFile, err := os.Create(wavFilePath)
	if err != nil {
		log.Debugf("Error creating wav file:%s", err)
		log.Errorf("Could not open wav file '%s'", wavFilePath)
	}

	writer := coremedia.NewAVFileWriter(bufio.NewWriter(h264File), bufio.NewWriter(wavFile))

	defer func() {
		stat, err := wavFile.Stat()
		if err != nil {
			log.Fatal("Could not get wav file stats", err)
		}
		err = coremedia.WriteWavHeader(int(stat.Size()), wavFile)
		if err != nil {
			log.Fatalf("Error writing wave header %s might be invalid. %s", wavFilePath, err.Error())
		}
		err = wavFile.Close()
		if err != nil {
			log.Fatalf("Error closing wave file. '%s' might be invalid. %s", wavFilePath, err.Error())
		}
		err = h264File.Close()
		if err != nil {
			log.Fatalf("Error closing h264File '%s'. %s", h264FilePath, err.Error())
		}

	}()
	startWithConsumer(writer, udid, false)
}

func startWithConsumer(consumer screencapture.CmSampleBufConsumer, udid string, audioOnly bool) {
	device, err := screencapture.FindIosDevice(udid)
	if err != nil {
		printErrJSON(err, "no device found to activate")
		return
	}

	device, err = screencapture.EnableQTConfig(device)
	if err != nil {
		printErrJSON(err, "Error enabling QT config")
		return
	}

	adapter := screencapture.UsbAdapter{}
	stopSignal := make(chan interface{})
	waitForSigInt(stopSignal)

	mp := screencapture.NewMessageProcessor(&adapter, stopSignal, consumer, audioOnly)

	err = adapter.StartReading(device, &mp, stopSignal)
	consumer.Stop()
	if err != nil {
		printErrJSON(err, "failed connecting to usb")
	}
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

func printErrJSON(err error, msg string) {
	printJSON(map[string]interface{}{
		"original_error": err.Error(),
		"error_message":  msg,
	})
}
func printJSON(output map[string]interface{}) {
	text, err := json.Marshal(output)
	if err != nil {
		log.Fatalf("Broken json serialization, error: %s", err)
	}
	println(string(text))
}
