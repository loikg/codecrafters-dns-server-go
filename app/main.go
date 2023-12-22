package main

import (
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
    return msg.Header.Serialize()
}

func (header DNSHeader) Serialize() []byte {
    buff := make([]byte, 12)
	var flags uint16

    binary.BigEndian.PutUint16(buff[:2], header.ID)

	if header.Flags.QR {
		flags |= 1 << 15 // set 1st bit
	}
    flags |= header.Flags.OPCODE << 11
	if header.Flags.AA {
		flags |= 1 << 10
	}
	if header.Flags.TC {
		flags |= 1 << 9
	}
	if header.Flags.RD {
		flags |= 1 << 8
	}
	if header.Flags.RA {
		flags |= 1 << 7
	}
    flags |= header.Flags.Z << 4
    flags |= header.Flags.RCODE

    binary.BigEndian.PutUint16(buff[2:4], flags)
    binary.BigEndian.PutUint16(buff[4:6], header.QDCOUNT)
    binary.BigEndian.PutUint16(buff[6:8], header.ANCOUNT)
    binary.BigEndian.PutUint16(buff[8:10], header.NSCOUNT)
    binary.BigEndian.PutUint16(buff[10:], header.ARCOUNT)

    return buff
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

        msg := DNSMessage{
            Header: DNSHeader{
                ID: 1234, 
                Flags: DNSHeaderFlags{
                    QR: true,
                },
            },
        }

		response := msg.Serialize()

        _, err = udpConn.WriteToUDP(response, source)
		if err != nil {
			fmt.Println("Failed to send response:", err)
		}
	}
}
