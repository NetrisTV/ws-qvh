package main

import (
	"encoding/binary"
	"github.com/danielpaulus/quicktime_video_hack/screencapture/coremedia"
)

var startCode = []byte{00, 00, 00, 01}

type NaluWriter struct {
	receiver *ReceiverHub
}

func NewNaluWriter(cliend *ReceiverHub) *NaluWriter {
	return &NaluWriter{receiver: cliend}
}

func (nw NaluWriter) consumeVideo(buf coremedia.CMSampleBuffer) error {
	if buf.HasFormatDescription {
		err := nw.writeNalu(buf.FormatDescription.PPS)
		if err != nil {
			return err
		}
		err = nw.writeNalu(buf.FormatDescription.SPS)
		if err != nil {
			return err
		}

	}
	if !buf.HasSampleData() {
		return nil
	}
	return nw.writeNalus(buf.SampleData)
}

func (nw NaluWriter) Consume(buf coremedia.CMSampleBuffer) error {
	if buf.MediaType == coremedia.MediaTypeSound {
		// we don't support audio for now
		//return nw.consumeAudio(buf)
		return nil
	}
	return nw.consumeVideo(buf)
}


func (nw NaluWriter) writeNalus(bytes []byte) error {
	slice := bytes
	for len(slice) > 0 {
		length := binary.BigEndian.Uint32(slice)
		err := nw.writeNalu(slice[4 : length+4])
		if err != nil {
			return err
		}
		slice = slice[length+4:]
	}
	return nil
}

func (nw NaluWriter) writeNalu(bytes []byte) error {
	if nw.receiver.closed {
		return nil
	}
	if len(bytes) > 0 {
		nw.receiver.send <- append(startCode, bytes...)
	}
	return nil
}

func (nw NaluWriter) Stop() {

}
