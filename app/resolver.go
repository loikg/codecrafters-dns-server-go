package main

import (
	"encoding/hex"
	"fmt"
	"log"
	"net"
	"time"
)

type Resolver struct {
	conn          *net.UDPConn
	serverUDPAddr *net.UDPAddr
}

func NewResolver(serverAddr string) (*Resolver, error) {
	serverUDPAddr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp", nil, serverUDPAddr)
	if err != nil {
		return nil, fmt.Errorf("failed to dial %v: %w", serverAddr, err)
	}

	return &Resolver{
		conn:          conn,
		serverUDPAddr: serverUDPAddr,
	}, nil
}

func (r Resolver) Close() error {
	return r.conn.Close()
}

func (r Resolver) SendRequest(msg *DNSMessage) (*DNSMessage, error) {
	log.Printf("sending resolve request to %s", r.serverUDPAddr)
	reqBuf, err := msg.MarshalBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to encode request: %v", err)
	}
	if _, err := r.write(reqBuf); err != nil {
		return nil, fmt.Errorf("failed to write request: %v", err)
	}

	resBuf := make([]byte, 512)
	n, _, err := r.readFromUDP(resBuf)
	if err != nil {
		return nil, err
	}
	res := &DNSMessage{}
	if err := res.UnmarshalBinary(resBuf[:n]); err != nil {
		return nil, err
	}

	log.Printf("received %d bytes: %s\n", len(resBuf[:n]), hex.EncodeToString(resBuf[:n]))
	log.Printf("resolve request response received from %s: %+v", r.serverUDPAddr, res)
	return res, nil
}

func (r Resolver) write(buf []byte) (int, error) {
	if err := r.conn.SetWriteDeadline(time.Now().Add(time.Second * 5)); err != nil {
		return 0, fmt.Errorf("failed to set udp connection write deadline: %v", err)
	}
	return r.conn.Write(buf)
}

func (r Resolver) readFromUDP(buf []byte) (int, *net.UDPAddr, error) {
	if err := r.conn.SetReadDeadline(time.Now().Add(time.Second * 5)); err != nil {
		return 0, nil, fmt.Errorf("failed to set udp connection read deadline: %v", err)
	}
	return r.conn.ReadFromUDP(buf)
}
