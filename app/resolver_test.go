package main

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net"
	"testing"
)

type testUDPServer struct {
	buf     bytes.Buffer
	stopped bool
	addr    string
}

func newTestUDPServer() *testUDPServer {
	return &testUDPServer{}
}

func (s *testUDPServer) Listen(startChan chan<- struct{}) {
	conn, err := net.ListenUDP("udp4", nil)
	if err != nil {
		panic(err)
	}
	fmt.Println(conn.LocalAddr(), conn.RemoteAddr())
	defer conn.Close()
	s.addr = conn.LocalAddr().String()
	startChan <- struct{}{}

	for !s.stopped {
		buf := make([]byte, 512)
		n, source, err := conn.ReadFromUDP(buf)
		if err != nil {
			panic(err)
		}
		if _, err = s.buf.Write(buf[:n]); err != nil {
			panic(err)
		}
		if _, err := conn.WriteToUDP(buf[:n], source); err != nil {
			panic(err)
		}
		s.stopped = true
	}
}

func (s *testUDPServer) Stop() {
	s.stopped = true
}

func TestResolver_SendRequest(t *testing.T) {
	startChan := make(chan struct{})
	testSrv := newTestUDPServer()
	req := DNSMessage{
		Header: DNSHeader{
			ID:      1234,
			Flags:   DNSHeaderFlags{},
			QDCOUNT: 1,
		},
	}
	req.AddQuestions(DNSQuestion{
		Name:  "google.com",
		Type:  1,
		Class: 1,
	})
	expectedBytes := []byte{
		0x04, 0xD2, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x06, 0x67, 0x6F, 0x6F, 0x67, 0x6C, 0x65, 0x03, 0x63, 0x6F, 0x6D, 0x00, 0x00, 0x01, 0x00, 0x01,
	}
	go testSrv.Listen(startChan)
	defer testSrv.Stop()
	<-startChan

	resolver, err := NewResolver(testSrv.addr)
	if err != nil {
		t.Fatalf("failed to create resolver: %v", err)
	}

	if _, err := resolver.SendRequest(&req); err != nil {
		t.Fatalf("failed to send request to resolver: %v", err)
	}
	if !bytes.Equal(expectedBytes, testSrv.buf.Bytes()) {
		t.Errorf(
			"expected to receive %s but got %s",
			hex.EncodeToString(expectedBytes),
			hex.EncodeToString(testSrv.buf.Bytes()),
		)
	}
}
