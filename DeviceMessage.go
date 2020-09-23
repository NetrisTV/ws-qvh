package main

import (
	"bytes"
	"encoding/binary"
)

const (
	Magic = "NtrsTV"
	TYPE_WDA_URL = 201
)

type DeviceMessage struct {
	bytes []byte
}

func NewWdaUrlMessage(code int, text []byte) DeviceMessage {
	var b bytes.Buffer
	b.WriteString(Magic)
	b.WriteByte(TYPE_WDA_URL)

	bs := make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(code))
	b.Write(bs)

	bs = make([]byte, 4)
	binary.BigEndian.PutUint32(bs, uint32(len(text)))
	b.Write(bs)

	b.Write(text)
	return DeviceMessage{ b.Bytes()}
}

func (d *DeviceMessage) Bytes() *[]byte {
	return &d.bytes
}
