package main

import (
	"bytes"
	"testing"
)

func TestDNSMessage_Serialize(t *testing.T) {
    tcs := []struct{
        name string
        msg DNSMessage
        expected []byte
    }{
        {
            name: "serialize header in correct byte binary representation",
            msg:    DNSMessage{
                Header: DNSHeader{
                    ID: 1234, 
                    Flags: DNSHeaderFlags{
                        QR: true,
                    },
                },
            },
            expected : []byte{0x04,0xD2, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
        },
        {
            name: "serialize question section in correct binary representation",
            msg:    DNSMessage{
                Header: DNSHeader{
                    ID: 1234, 
                    Flags: DNSHeaderFlags{
                        QR: true,
                    },
                    QDCOUNT: 1,
                },
                Questions: []DNSQuestion{
                    {
                        Name: "google.com" ,
                        Type: 0x01,
                        Class: 0x01,
                    },
                },
            },
            expected : []byte{
                0x04,0xD2, 0x80, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Header
                0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, // Question
                0x00, 0x01, 0x00, 0x01,
            },
        },
        {
            name: "serialize answer section in correct binary representation",
            msg:    DNSMessage{
                Header: DNSHeader{
                    ID: 1234, 
                    Flags: DNSHeaderFlags{
                        QR: true,
                    },
                    ANCOUNT: 1,
                },
                Questions: []DNSQuestion{
                    {
                        Name: "google.com" ,
                        Type: 0x01,
                        Class: 0x01,
                    },
                },
                Answers: []DNSAnswer{
                    {
                        Name: "google.com",
                        Type: 1,
                        Class: 1,
                        TTL: 60,
                        Data: []byte{0x8, 0x8, 0x8, 0x8},
                    },
                },
            },
            expected : []byte{
                // Header
                0x04,0xD2, 0x80, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
                // Questions
                0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00,
                0x00, 0x01, 0x00, 0x01,
                // Answers
                0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00,
                0x0, 0x01, 0x0, 0x01, 0x0, 0x0, 0x0, 0x3C, 0x0, 0x04, 0x08, 0x08, 0x08,
                0x08,
            },
        },
    }


    for _, tc := range tcs {
        tc := tc
        t.Run(tc.name, func(t *testing.T) {
            msg := tc.msg.Serialize()
            //t.Logf("%s => %s\n", hex.EncodeToString(tc.expected), hex.EncodeToString(msg))
            t.Log(len(msg), len(tc.expected))
            if !bytes.Equal(tc.expected, msg) {
                t.Errorf("expected %x but got %x\n", tc.expected, msg)
            }
        })
    }
}
