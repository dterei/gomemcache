package main

// Encodes the binary wire protocol of memcache.

import (
	"encoding/binary"
	"io"
)

// MAX_VALUE_SIZE represents the largest key-value pair we will store.
const MAX_VALUE_SIZE = 1024 * 1024

// Protocol represents the memcache protocol used for a client connection.
type Protocol uint8

// List of memcache protocols available.
const (
	PROTOCOL_BINARY Protocol = iota
	PROTOCOL_ASCII
	PROTOCOL_AUTO
)

// Command represents a memcache command on the wire.
type Command uint8

// List of memcache commands.
const (
	CMD_GET             = Command(0x00)
	CMD_SET             = Command(0x01)
	CMD_ADD             = Command(0x02)
	CMD_REPLACE         = Command(0x03)
	CMD_DELETE          = Command(0x04)
	CMD_INCREMENT       = Command(0x05)
	CMD_DECREMENT       = Command(0x06)
	CMD_QUIT            = Command(0x07)
	CMD_FLUSH           = Command(0x08)
	CMD_GETQ            = Command(0x09)
	CMD_NOOP            = Command(0x0a)
	CMD_VERSION         = Command(0x0b)
	CMD_GETK            = Command(0x0c)
	CMD_GETKQ           = Command(0x0d)
	CMD_APPEND          = Command(0x0e)
	CMD_PREPEND         = Command(0x0f)
	CMD_STAT            = Command(0x10)
	CMD_SETQ            = Command(0x11)
	CMD_ADDQ            = Command(0x12)
	CMD_REPLACEQ        = Command(0x13)
	CMD_DELETEQ         = Command(0x14)
	CMD_INCREMENTQ      = Command(0x15)
	CMD_DECREMENTQ      = Command(0x16)
	CMD_QUITQ           = Command(0x17)
	CMD_FLUSHQ          = Command(0x18)
	CMD_APPENDQ         = Command(0x19)
	CMD_PREPENDQ        = Command(0x1a)
	CMD_VERBOSITY       = Command(0x1b) // memcache doesn't implement, but in spec.
	CMD_TOUCH           = Command(0x1c)
	CMD_GAT             = Command(0x1d)
	CMD_GATQ            = Command(0x1e)
	CMD_GATK            = Command(0x23)
	CMD_GATKQ           = Command(0x24)
	CMD_SASL_LIST_MECHS = Command(0x20)
	CMD_SASL_AUTH       = Command(0x21)
	CMD_SASL_STEP       = Command(0x22)
)

// Status represents a memcache status response code.
type Status uint16

// memcache status codes.
const (
	STATUS_OK               = Status(0x00)
	STATUS_KEY_NOT_FOUND    = Status(0x01)
	STATUS_KEY_EXISTS       = Status(0x02)
	STATUS_VALUE_TOO_LARGE  = Status(0x03)
	STATUS_INVALID_ARGUMENT = Status(0x04)
	STATUS_ITEM_NOT_STORED  = Status(0x05)
	STATUS_NON_NUMERIC      = Status(0x06)
	STATUS_AUTH_FAILED      = Status(0x20)
	STATUS_UNKNOWN_COMMAND  = Status(0x81)
	STATUS_OUT_OF_MEMORY    = Status(0x82)
	STATUS_BUSY             = Status(0x85)
)

// RequestType is the type of memcache request.
type RequestType uint8

const (
	// MSG_REQUEST is the magic code for memcache request type.
	MSG_REQUEST  = RequestType(0x80)

	// MSG_RESPONSE is the magic code for memcache response type.
	MSG_RESPONSE = RequestType(0x81)
)

// Header is a memcache request/response header.
type Header struct {
	Magic        RequestType
	Opcode       Command
	KeyLength    uint16
	ExtrasLength uint8
	DataType     uint8
	Status       uint16
	TotalLength  uint32
	Opaque       uint32
	Cas          uint64
}

// HEADER_SIZE is the size in bytes of a memcache header.
const HEADER_SIZE = 24

// BodyLength returns the expected body length of a memcache request.
func (hdr *Header) BodyLength() uint32 {
	return hdr.TotalLength - (uint32)(hdr.ExtrasLength) - (uint32)(hdr.KeyLength)
}

// HeaderParseError represents an invalid header from a client.
type HeaderParseError struct{}

func (*HeaderParseError) Error() string {
	return "Failed to parse header"
}

// ReadRequest reads a memcache request header from a stream.
func (hdr *Header) ReadRequest(conn io.Reader) error {
	buf := make([]byte, HEADER_SIZE)
	_, err := io.ReadFull(conn, buf)
	if err != nil {
		return err
	}

	hdr.Magic = RequestType(buf[0])
	hdr.Opcode = Command(buf[1])
	hdr.KeyLength = binary.BigEndian.Uint16(buf[2:])
	hdr.ExtrasLength = buf[4]
	hdr.DataType = buf[5]
	hdr.Status = binary.BigEndian.Uint16(buf[6:])
	hdr.TotalLength = binary.BigEndian.Uint32(buf[8:])
	hdr.Opaque = binary.BigEndian.Uint32(buf[12:])
	hdr.Cas = binary.BigEndian.Uint64(buf[16:])

	return hdr.CheckValidRequest()
}

// CheckValidRequest checks a memcache header is a valid request header.
func (hdr *Header) CheckValidRequest() error {
	if hdr.Magic != MSG_REQUEST {
		return &HeaderParseError{}
	} else if uint32(hdr.KeyLength)+uint32(hdr.ExtrasLength) > hdr.TotalLength {
		return &HeaderParseError{}
	}
	return nil
}

// NewResponse creates a Memcache response message header.
func NewResponse(command Command,
	status Status,
	key []byte,
	value []byte,
	extras []byte,
	opaque uint32,
	version uint64) Header {
	return Header{
		Magic:        MSG_RESPONSE,
		Opcode:       command,
		KeyLength:    uint16(len(key)),
		ExtrasLength: uint8(len(extras)),
		DataType:     0,
		Status:       uint16(status),
		TotalLength:  uint32(len(value) + len(extras) + len(key)),
		Opaque:       opaque,
		Cas:          version,
	}
}

// Serialize writes the header out to the buffer provided. The buffer should
// have length HEADER_SIZE.
func (hdr *Header) Serialize(buf []byte) {
	buf[0] = uint8(hdr.Magic)
	buf[1] = uint8(hdr.Opcode)
	binary.BigEndian.PutUint16(buf[2:], hdr.KeyLength)
	buf[4] = hdr.ExtrasLength
	buf[5] = hdr.DataType
	binary.BigEndian.PutUint16(buf[6:], hdr.Status)
	binary.BigEndian.PutUint32(buf[8:], hdr.TotalLength)
	binary.BigEndian.PutUint32(buf[12:], hdr.Opaque)
	binary.BigEndian.PutUint64(buf[16:], hdr.Cas)
}

// WriteTo writes out a memcache header.
func (hdr *Header) WriteTo(conn io.Writer) (int64, error) {
	buf := make([]byte, HEADER_SIZE)
	hdr.Serialize(buf)
	n, err := conn.Write(buf)
	return int64(n), err
}

// WriteResponse writes out a complete memcache response.
func WriteResponse(conn io.Writer, hdr *Header, extras, key, value []byte) error {
	_, err := hdr.WriteTo(conn)
	if err != nil {
		return err
	}
	if extras != nil {
		_, err = conn.Write(extras)
		if err != nil {
			return err
		}
	}
	if key != nil {
		_, err = conn.Write(key)
		if err != nil {
			return err
		}
	}
	if value != nil {
		_, err = conn.Write(value)
		if err != nil {
			return err
		}
	}
	return nil
}
