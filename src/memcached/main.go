package main

// Run the memcache server on port 11211 with a 100MB storage limit.

import (
	"io/ioutil"
	"log"
	"net"
)

func init() {
	// Disabled log output normally, but comment out for testing
	// TODO: Better log facility like seelog.
	log.SetOutput(ioutil.Discard)
}

func main() {
	addr, err := net.ResolveTCPAddr("tcp", ":11211")
	if err != nil {
		log.Fatalf("ERROR: Cannot parse listen address: %s\n", err)
	}
	cache := NewCache(1024 * 1024 * 100)
	handler := NewConnectionHandler(cache, addr)
	handler.Run()
}
