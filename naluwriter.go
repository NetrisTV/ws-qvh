package main

import (
	"encoding/binary"
	"github.com/danielpaulus/quicktime_video_hack/screencapture/coremedia"
	log "github.com/sirupsen/logrus"
)

var startCode = []byte{00, 00, 00, 01}

type NaluWriter struct {
	receiver *ReceiverHub
}

func NewNaluHubWriter(cliend *ReceiverHub) NaluWriter {
	return NaluWriter{receiver: cliend}
}

func (nhw NaluWriter) consumeVideo(buf coremedia.CMSampleBuffer) error {
	if buf.HasFormatDescription {
		log.Info("PPS " + buf.FormatDescription.String())
		err := nhw.writeNalu(buf.FormatDescription.PPS)
		if err != nil {
			return err
		}
		err = nhw.writeNalu(buf.FormatDescription.SPS)
		if err != nil {
			return err
		}
	}
	if !buf.HasSampleData() {
		return nil
	}
	return nhw.writeNalus(buf.SampleData)
}

func (nhw NaluWriter) Consume(buf coremedia.CMSampleBuffer) error {
	if buf.MediaType == coremedia.MediaTypeSound {
		//return nhw.consumeAudio(buf)
		return nil
	}
	return nhw.consumeVideo(buf)
}


func (nhw NaluWriter) writeNalus(bytes []byte) error {
	slice := bytes
	for len(slice) > 0 {
		length := binary.BigEndian.Uint32(slice)
		err := nhw.writeNalu(slice[4 : length+4])
		if err != nil {
			return err
		}
		slice = slice[length+4:]
	}
	return nil
}

func (nhw NaluWriter) writeNalu(bytes []byte) error {
	if nhw.receiver.closed {
		return nil
	}
	if len(bytes) > 0 {
		nhw.receiver.send <- append(startCode, bytes...)
	}
	return nil
}

func (nhw NaluWriter) Stop() {

}
