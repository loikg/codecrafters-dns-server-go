package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"net"
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

	udpAddr, err := net.ResolveUDPAddr("udp", "127.0.0.1:2053")
	if err != nil {
		log.Fatalf("Failed to resolve UDP address: %v", err)
	}

	resolver, err := NewResolver(resolverAddress)
	if err != nil {
		log.Printf("failed to create resolver for address %s: %v", resolverAddress, err)
		return
	}

	udpConn, err := net.ListenUDP("udp", udpAddr)
	if err != nil {
		log.Fatalf("Failed to bind to address: %v", err)
	}
	defer udpConn.Close()

	buf := make([]byte, 512)

	for {
		size, source, err := udpConn.ReadFromUDP(buf)
		if err != nil {
			log.Printf("Error receiving data: %v", err)
			continue
		}

		log.Printf("processing request from %s: %s", source, hex.EncodeToString(buf[:size]))

		req := DNSMessage{}
		if err := req.UnmarshalBinary(buf[:size]); err != nil {
			log.Printf("failed to parse request: %v", err)
			return
		}
		log.Printf("REQ: %+v\n", req)
		resp, err := processMessage(resolver, &req)
		if err != nil {
			log.Printf("failed to process message: %v", err)
			continue
		}
		response, err := resp.MarshalBinary()
		if err != nil {
			log.Printf("failed to marshal respoinse: %v\n", err)
			continue
		}

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			log.Println("Failed to send response:", err)
			continue
		}
		log.Printf("request processed %s", source)
	}
}

func processMessage(resolver *Resolver, req *DNSMessage) (*DNSMessage, error) {
	resp := CreateResponse(req)

	for i, q := range req.Questions {
		req := DNSMessage{
			Header: DNSHeader{
				ID:      uint16(i),
				Flags:   DNSHeaderFlags{},
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
			return nil, fmt.Errorf("failed to send resolve request: %v", err)
		}
		resp.AddAnswers(r.Answers[0])
		log.Printf("resolver request %d successfull", i)
	}

	return resp, nil
}
