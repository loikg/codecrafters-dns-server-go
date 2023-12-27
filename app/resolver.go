package main

import (
	"fmt"
	"net"
)

type Resolver struct {
	conn          *net.UDPConn
	serverUDPAddr *net.UDPAddr
}

func NewResolver(serverAddr string) (*Resolver, error) {
	serverUDPAddr, err := net.ResolveUDPAddr("udp4", serverAddr)
	if err != nil {
		return nil, err
	}
	conn, err := net.DialUDP("udp4", nil, serverUDPAddr)
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
	reqBuf, err := msg.MarshalBinary()
	if err != nil {
		return nil, err
	}
	if _, err := r.conn.Write(reqBuf); err != nil {
		return nil, err
	}

	resBuf := make([]byte, 512)
	if _, _, err := r.conn.ReadFromUDP(resBuf); err != nil {
		return nil, err
	}
	res := &DNSMessage{}
	if err := res.UnmarshalBinary(resBuf); err != nil {
		return nil, err
	}

	return res, nil
}
