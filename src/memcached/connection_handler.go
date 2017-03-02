package main

import (
	"log"
	"net"
)

// ConnectionHandler handles accepting new connections from clients and serving
// their requests.
type ConnectionHandler struct {
	cache        *Cache
	listener     *net.TCPListener
	totalClients uint
}

// NewConnectionHandler creates a new ConnectionHandler to accept incoming
// memcache connections and run them against the specificed Cache.
func NewConnectionHandler(cache *Cache, addr *net.TCPAddr) *ConnectionHandler {
	l, err := net.ListenTCP("tcp", addr)
	if err != nil {
		log.Fatalf("ERROR: Cannot listen: %s\n", err)
	}
	log.Printf("INFO: Listenning on: %s\n", l.Addr())

	return &ConnectionHandler{cache, l, 0}
}

// Run runs the ConnectionHandler, it never returns.
func (cnh *ConnectionHandler) Run() {
	for {
		conn, err := cnh.listener.AcceptTCP()
		if err != nil {
			log.Printf("ERROR: Accept: %s\n", err)
			continue
		}
		cnh.runClient(conn)
	}
}

// runClient manages a new client connection.
func (cnh *ConnectionHandler) runClient(conn *net.TCPConn) {
	client := NewClientConn(cnh.totalClients, cnh.cache, conn)
	cnh.totalClients++
	go client.Run()
}
