package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math/rand"
	"time"
)

const (
	FrameVersion byte = 1

	FrameTypeNew   byte = 1
	FrameTypeData  byte = 2
	FrameTypeFin   byte = 3
	FrameTypeError byte = 4

	MaxPayloadSize = 1200 // keep < MTU
)

type Frame struct {
	Version byte
	Type    byte
	ConnID  uint32
	Payload []byte
}

// Encode a frame into bytes (with random padding for obfuscation)
func (f *Frame) Encode() ([]byte, error) {
	var buf bytes.Buffer

	// Header: version(1) | type(1) | connID(4) | length(2)
	if err := buf.WriteByte(f.Version); err != nil {
		return nil, err
	}
	if err := buf.WriteByte(f.Type); err != nil {
		return nil, err
	}
	if err := binary.Write(&buf, binary.BigEndian, f.ConnID); err != nil {
		return nil, err
	}

	payloadLen := uint16(len(f.Payload))
	if err := binary.Write(&buf, binary.BigEndian, payloadLen); err != nil {
		return nil, err
	}

	// Payload
	if _, err := buf.Write(f.Payload); err != nil {
		return nil, err
	}

	// Padding: add random junk up to MTU-safe size
	padMax := MaxPayloadSize - buf.Len()
	if padMax > 0 {
		rand.Seed(time.Now().UnixNano())
		padLen := rand.Intn(padMax)
		padding := make([]byte, padLen)
		rand.Read(padding)
		buf.Write(padding)
	}

	return buf.Bytes(), nil
}

// Decode bytes into a Frame
func DecodeFrame(data []byte) (*Frame, error) {
	if len(data) < 8 { // minimum header size
		return nil, fmt.Errorf("frame too short")
	}

	f := &Frame{}
	f.Version = data[0]
	f.Type = data[1]
	f.ConnID = binary.BigEndian.Uint32(data[2:6])
	payloadLen := binary.BigEndian.Uint16(data[6:8])

	if int(payloadLen) > len(data)-8 {
		return nil, fmt.Errorf("invalid payload length")
	}

	f.Payload = make([]byte, payloadLen)
	copy(f.Payload, data[8:8+payloadLen])

	return f, nil
}
