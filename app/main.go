package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
)

type DNSMessage struct {
    Header DNSHeader
}

type DNSHeader struct {
	ID      uint16
	Flags   DNSHeaderFlags
	QDCOUNT uint16
	ANCOUNT uint16
	NSCOUNT uint16
	ARCOUNT uint16
}

type DNSHeaderFlags struct {
	QR     bool
	OPCODE uint16 // 4bit
	AA     bool
	TC     bool
	RD     bool
	RA     bool
	Z      uint16 // 3bit
	RCODE  uint16 // 4bit
}

func (msg DNSMessage) Serialize() []byte {
    var buff bytes.Buffer

    binary.Write(&buff, binary.BigEndian, msg.Header)

    return buff.Bytes()
}

func (header DNSHeader) Serialize() []byte {
	var buff bytes.Buffer
	var fields uint16
	binary.Write(&buff, binary.BigEndian, header.ID)

	if header.Flags.QR {
		fields |= 1 << 15 // set 1st bit
	}
    fields |= header.Flags.OPCODE << 11
	if header.Flags.AA {
		fields |= 1 << 10
	}
	if header.Flags.TC {
		fields |= 1 << 9
	}
	if header.Flags.RD {
		fields |= 1 << 8
	}
	if header.Flags.RA {
		fields |= 1 << 7
	}
    fields |= header.Flags.Z << 4
    fields |= header.Flags.RCODE << 0

    binary.Write(&buff, binary.BigEndian, fields)
	binary.Write(&buff, binary.BigEndian, header.QDCOUNT)
	binary.Write(&buff, binary.BigEndian, header.ANCOUNT)
	binary.Write(&buff, binary.BigEndian, header.NSCOUNT)
	binary.Write(&buff, binary.BigEndian, header.ARCOUNT)
	return buff.Bytes()
}

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

		response := DNSMessage{
            Header: DNSHeader{
                ID: 1234, 
                Flags: DNSHeaderFlags{
                    QR: true,
                },
            },
        }.Serialize()

		_, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
