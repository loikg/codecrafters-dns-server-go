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
                0x04,0xD2, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Header
                0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, // Question
                0x00, 0x01, 0x00, 0x01,
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
