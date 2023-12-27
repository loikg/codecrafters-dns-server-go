package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
)

func main() {
	resolver := flag.String(
		"resolver",
		"r",
		"address of DNS resolver to forward requests to: 0.0.0.0:53",
	)

	flag.Parse()
	fmt.Println(resolver)

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

		fmt.Printf("REQ: %s\n", hex.EncodeToString(buf[:size]))
		req := DNSMessage{}
		if err := req.UnmarshalBinary(buf[:size]); err != nil {
			fmt.Println(err)
		}
		fmt.Printf("REQ: %+v\n", req)

		resp := CreateResponse(req)

		for _, q := range req.Questions {
			resp.AddAnswers(DNSAnswer{
				Name:  q.Name,
				Type:  q.Type,
				Class: q.Class,
				TTL:   60,
				Data:  []byte{0x8, 0x8, 0x8, 0x8},
			})
		}

		fmt.Printf("RESP: %+v\n", resp)

		response, err := resp.MarshalBinary()
		if err != nil {
			fmt.Printf("failed to marshal respoinse: %v\n", err)
			continue
		}

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
