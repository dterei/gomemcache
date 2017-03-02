package main

import (
	"bytes"
	"encoding/binary"
	"testing"
)

func TestHeaderSerialize(t *testing.T) {
	bts := make([]byte, 24)
	originalHeader := Header{
		Magic:        0x81,
		Status:       0x4,
		KeyLength:    16,
		ExtrasLength: 4,
		DataType:     3,
		TotalLength:  1024,
		Opaque:       1334,
		Cas:          8839298347234234,
	}

	originalHeader.Serialize(bts)

	buf := new(bytes.Buffer)
	buf.Write(bts)
	var header Header
	binary.Read(buf, binary.BigEndian, &header)

	if header != originalHeader {
		t.Errorf("Expected %v but was %v", originalHeader, header)
	}
}

func TestHeaderParse(t *testing.T) {
	buf := new(bytes.Buffer)
	originalHeader := Header{
		Magic:        0x81,
		Status:       0x4,
		KeyLength:    16,
		ExtrasLength: 4,
		DataType:     3,
		TotalLength:  1024,
		Opaque:       1334,
		Cas:          8839298347234234,
	}
	binary.Write(buf, binary.BigEndian, &originalHeader)

	bts := buf.Bytes()

	var header Header
	header.ReadRequest(bytes.NewReader(bts))

	if header != originalHeader {
		t.Errorf("Expected %v but was %v", originalHeader, header)
	}
}
