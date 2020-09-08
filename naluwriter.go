package main

import (
	"encoding/binary"
	"github.com/danielpaulus/quicktime_video_hack/screencapture/coremedia"
	log "github.com/sirupsen/logrus"
)

var startCode = []byte{00, 00, 00, 01}

type NaluHubWriter struct {
	client *Client
}

func NewNaluHubWriter(cliend *Client) NaluHubWriter {
	return NaluHubWriter{client: cliend}
}

func (nhw NaluHubWriter) consumeVideo(buf coremedia.CMSampleBuffer) error {
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

func (nhw NaluHubWriter) Consume(buf coremedia.CMSampleBuffer) error {
	if buf.MediaType == coremedia.MediaTypeSound {
		//return nhw.consumeAudio(buf)
		return nil
	}
	return nhw.consumeVideo(buf)
}


func (nhw NaluHubWriter) writeNalus(bytes []byte) error {
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

func (nhw NaluHubWriter) writeNalu(bytes []byte) error {
	if nhw.client.closed {
		return nil
	}
	if len(bytes) > 0 {
		nhw.client.send <- append(startCode, bytes...)
	}
	return nil
}

func (nhw NaluHubWriter) Stop() {

}
