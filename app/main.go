package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"net"
	"strings"
)

type DNSMessage struct {
    Header DNSHeader
    Questions []DNSQuestion
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

type DNSQuestion struct {
    Name string
    Type uint16
    Class uint16
}

func (q DNSQuestion) Serialize() []byte {
    var buff bytes.Buffer

    serializedDomain := serializeDomainName(q.Name)
    if _, err := buff.Write(serializedDomain); err != nil {
        panic(err)
    }
    if err := binary.Write(&buff, binary.BigEndian, q.Type); err != nil {
        panic(err)
    }
    if err := binary.Write(&buff, binary.BigEndian, q.Class); err != nil {
        panic(err)
    }

    return buff.Bytes()[:buff.Len()]
}

func (msg DNSMessage) Serialize() []byte {
    var buff bytes.Buffer
    buff.Write(msg.Header.Serialize())
    for _, q := range msg.Questions {
        buff.Write(q.Serialize())
    }

    return buff.Bytes()[:buff.Len()]
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
                QDCOUNT: 1,
            },
            Questions: []DNSQuestion{
                {
                    //Name: "google.com",
                    Name: "codecrafters.io",
                    Type: 0x0001,
                    Class: 0x0001,
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

func serializeDomainName(domain string) []byte {
    var buff bytes.Buffer

    labels := strings.Split(domain, ".")
    for _, label := range labels {
        b := []byte(label)
        buff.WriteByte(uint8(len(b)))
        buff.Write(b)
    }
    buff.WriteByte(0x0)

    return buff.Bytes()[:buff.Len()]
}
