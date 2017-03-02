package main

import (
	"bufio"
	"io"
	"log"
	"net"
)

// ClientConn represents a connection with a single memcache client.
type ClientConn struct {
	id    uint
	cache *Cache
	conn  *net.TCPConn
	bio   *bufio.ReadWriter
}

// NewClientConn creates a new ClientConn to manage the TCP connection for a
// memcache client.
func NewClientConn(id uint, cache *Cache, conn *net.TCPConn) *ClientConn {
	conn.SetLinger(0)
	conn.SetKeepAlive(true)
	conn.SetNoDelay(true)
	bio := bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn))
	return &ClientConn{id, cache, conn, bio}
}

// Run loops forever, processing a client connection for incoming requests.
func (client *ClientConn) Run() {
	defer client.conn.Close()
	defer client.bio.Flush()
	defer log.Printf("INFO: [%d] End client\n", client.id)

	log.Printf("INFO: [%d] New client\n", client.id)

	var req Header

	for {
		// read header
		err := req.ReadRequest(client.bio)
		if err != nil {
			if err != io.ErrUnexpectedEOF && err != io.EOF {
				log.Printf("ERROR: [%d] Reading header: %s\n", client.id, err)
			}
			return
		}

		// validate size - we could perhaps get away with a far larger size as we
		// aren't using a hand-rolled slab allocator like memcached, but a max size
		// to ensure some safety (e.g., no 4GB value) is reasonable.
		if req.TotalLength > MAX_VALUE_SIZE {
			resp := NewResponse(req.Opcode, STATUS_VALUE_TOO_LARGE,
				nil, nil, nil, req.Opaque, 0)
			WriteResponse(client.bio, &resp, nil, nil, nil)
			return
		}

		// read body
		body := make([]byte, req.TotalLength)
		_, err = io.ReadFull(client.bio, body)
		if err != nil {
			log.Printf("ERROR: [%d] Reading body: %s\n", client.id, err)
			return
		}
		extras := body[:req.ExtrasLength]
		key := body[req.ExtrasLength:][:req.KeyLength]
		value := body[req.ExtrasLength:][req.KeyLength:]

		// run command
		switch req.Opcode {
		case CMD_GET:
			err = client.handleGet(&req, extras, key, value)
		case CMD_SET:
			err = client.handleSet(&req, extras, key, value)
		case CMD_DELETE:
			err = client.handleDelete(&req, extras, key, value)
		default:
			resp := NewResponse(req.Opcode, STATUS_UNKNOWN_COMMAND,
				nil, nil, nil, req.Opaque, 0)
			WriteResponse(client.bio, &resp, nil, nil, nil)
		}

		// flush output
		client.bio.Flush()
		if err != nil {
			return
		}
	}
}

// handleGet handles the memcache get command.
func (client *ClientConn) handleGet(req *Header, extras, key, value []byte) error {
	log.Printf("INFO: [%d] - get\n", client.id)

	if len(extras) != 0 || len(value) != 0 {
		resp := NewResponse(req.Opcode, STATUS_INVALID_ARGUMENT,
			nil, nil, nil, req.Opaque, 0)
		return WriteResponse(client.bio, &resp, nil, nil, nil)
	}

	item := client.cache.Get(key)

	if item == nil {
		resp := NewResponse(req.Opcode, STATUS_KEY_NOT_FOUND, nil, nil, nil, req.Opaque, 0)
		return WriteResponse(client.bio, &resp, nil, nil, nil)
	}

	resp := NewResponse(req.Opcode, STATUS_OK, nil, item.value, item.flags[:], req.Opaque, item.version)
	return WriteResponse(client.bio, &resp, item.flags[:], nil, item.value)
}

// handleSet handles the memcache set command.
func (client *ClientConn) handleSet(req *Header, extras, key, value []byte) error {
	log.Printf("INFO: [%d] - set\n", client.id)

	if len(extras) != 8 || len(value) == 0 {
		resp := NewResponse(req.Opcode, STATUS_INVALID_ARGUMENT,
			nil, nil, nil, req.Opaque, 0)
		return WriteResponse(client.bio, &resp, nil, nil, nil)
	}

	ver, ok := client.cache.Set(key, value, extras[0:4], req.Cas)

	resp := NewResponse(req.Opcode, ok, nil, nil, nil, req.Opaque, ver)
	return WriteResponse(client.bio, &resp, nil, nil, nil)
}

// handleDelete handles the memcache delete command.
func (client *ClientConn) handleDelete(req *Header, extras, key, value []byte) error {
	log.Printf("INFO: [%d] - delete\n", client.id)

	if len(extras) != 0 || len(value) != 0 {
		resp := NewResponse(req.Opcode, STATUS_INVALID_ARGUMENT,
			nil, nil, nil, req.Opaque, 0)
		return WriteResponse(client.bio, &resp, nil, nil, nil)
	}

	ok := client.cache.Delete(key, req.Cas)

	resp := NewResponse(req.Opcode, ok, nil, nil, nil, req.Opaque, 0)
	return WriteResponse(client.bio, &resp, nil, nil, nil)
}
