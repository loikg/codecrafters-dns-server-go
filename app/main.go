package main

import (
	"fmt"
	"net"
)

func main() {
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

		receivedData := string(buf[:size])
		fmt.Printf("Received %d bytes from %s: %s\n", size, source, receivedData)

		req := DNSMessage{}
		if err := req.UnmarshalBinary(buf); err != nil {
			fmt.Println(err)
		}

		fmt.Printf("REQ: %x\n %+v\n", buf, req)

		resp := CreateResponse(req)

		resp.AddQuestions(DNSQuestion{

			Name:  "codecrafters.io",
			Type:  0x0001,
			Class: 0x0001,
		})
		resp.AddAnswers(DNSAnswer{
			Name:  "codecrafters.io",
			Type:  1,
			Class: 1,
			TTL:   60,
			Data:  []byte{0x8, 0x8, 0x8, 0x8},
		})

		if req.Header.Flags.OPCODE != 0 {
			resp.Header.Flags.RCODE = 4
		}

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
