package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
)

var resolverAddress string

func main() {
	flag.StringVar(
		&resolverAddress,
		"resolver",
		"8.8.8.8:53",
		"address of DNS resolver to forward requests to: 0.0.0.0:53",
	)

	flag.Parse()

	fmt.Println("resolverAddress: ", resolverAddress)

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		fmt.Println("Failed to resolve UDP address:", err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		fmt.Println("Failed to bind to address:", err)
		return
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			fmt.Println("Error receiving data:", err)
			break
		}

		reqBuf := make([]byte, size)
		copy(reqBuf, buf)
		go processMessage(udpConn, source, reqBuf)
	}
}

func processMessage(conn *net.UDPConn, source *net.UDPAddr, buf []byte) {
	// foreach question
	//      Create and encode DNSMessage
	//      Forward to resolve
	// construct answer
	log.Printf("processing request from %s", source)
	log.Printf("REQ: %s", hex.EncodeToString(buf))
	req := DNSMessage{}
	if err := req.UnmarshalBinary(buf); err != nil {
		fmt.Println(err)
	}
	fmt.Printf("REQ: %+v\n", req)

	resp := CreateResponse(req)

	var wg sync.WaitGroup

	answers := make([]DNSAnswer, len(req.Questions))
	for i, q := range req.Questions {
		wg.Add(1)
		go func(i int, q DNSQuestion) {
			defer wg.Done()
			resolver, err := NewResolver(resolverAddress)
			if err != nil {
				log.Printf("failed to create resolver for address %s: %v", resolverAddress, err)
				return
			}
			defer resolver.Close()
			req := DNSMessage{
				Header: DNSHeader{
					ID: uint16(i),
					Flags: DNSHeaderFlags{
						QR: true,
					},
					QDCOUNT: 1,
				},
				Questions: []DNSQuestion{
					{
						Name:  q.Name,
						Type:  q.Type,
						Class: q.Class,
					},
				},
			}
			r, err := resolver.SendRequest(&req)
			if err != nil {
				log.Printf("resolver request failed: %v", err)
				return
			}
			answers[i] = r.Answers[0]
		}(i, q)
	}

	wg.Wait()

	resp.AddAnswers(answers...)
	fmt.Printf("RESP: %+v\n", resp)

	response, err := resp.MarshalBinary()
	if err != nil {
		fmt.Printf("failed to marshal respoinse: %v\n", err)
		return
	}

	_, err = conn.WriteToUDP(response, source)
	if err != nil {
		fmt.Println("Failed to send response:", err)
	}
	log.Printf("request processed %s", source)
}
