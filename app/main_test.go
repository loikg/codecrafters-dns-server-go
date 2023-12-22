package main

import (
    "slices"
    "testing"
)

func TestDNSMessage_Serialize(t *testing.T) {
    expected := []byte{0x04,0xD2, 0x10, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00}

    msg := DNSMessage{
        Header: DNSHeader{
            ID: 1234, 
            Flags: DNSHeaderFlags{
                QR: true,
            },
        },
    }.Serialize()

    t.Logf("%b => %b\n", expected, msg)

    if slices.Equal(expected, msg) {
        t.Errorf("extected %b but got %b\n", expected, msg)
    }
}
