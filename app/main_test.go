package main

import (
	"bytes"
	"io"
	"reflect"
	"testing"
)

func TestDNSMessage_Serialize(t *testing.T) {
	tcs := []struct {
		name     string
		msg      DNSMessage
		expected []byte
	}{
		{
			name: "serialize header in correct byte binary representation",
			msg: DNSMessage{
				Header: DNSHeader{
					ID: 1234,
					Flags: DNSHeaderFlags{
						QR: true,
					},
				},
			},
			expected: []byte{0x04, 0xD2, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		},
		{
			name: "serialize question section in correct binary representation",
			msg: DNSMessage{
				Header: DNSHeader{
					ID: 1234,
					Flags: DNSHeaderFlags{
						QR: true,
					},
					QDCOUNT: 1,
				},
				Questions: []DNSQuestion{
					{
						Name:  "google.com",
						Type:  0x01,
						Class: 0x01,
					},
				},
			},
			expected: []byte{
				0x04, 0xD2, 0x80, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Header
				0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, // Question
				0x00, 0x01, 0x00, 0x01,
			},
		},
		{
			name: "serialize answer section in correct binary representation",
			msg: DNSMessage{
				Header: DNSHeader{
					ID: 1234,
					Flags: DNSHeaderFlags{
						QR: true,
					},
					ANCOUNT: 1,
				},
				Questions: []DNSQuestion{
					{
						Name:  "google.com",
						Type:  0x01,
						Class: 0x01,
					},
				},
				Answers: []DNSAnswer{
					{
						Name:  "google.com",
						Type:  1,
						Class: 1,
						TTL:   60,
						Data:  []byte{0x8, 0x8, 0x8, 0x8},
					},
				},
			},
			expected: []byte{
				// Header
				0x04, 0xD2, 0x80, 0x00, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
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
			msg, err := tc.msg.MarshalBinary()
			if err != nil {
				t.Fatalf("expected no error got %v", err)
			}
			//t.Logf("%s => %s\n", hex.EncodeToString(tc.expected), hex.EncodeToString(msg))
			t.Log(len(msg), len(tc.expected))
			if !bytes.Equal(tc.expected, msg) {
				t.Errorf("expected %x but got %x\n", tc.expected, msg)
			}
		})
	}
}

func TestUnMarshalDomain(t *testing.T) {
	expected := "google.com"
	buf := []byte{0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00}

	result, err := UnMarshalDomain(buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result != expected {
		t.Errorf("expected: %s but got %s", expected, result)
	}
}

func TestReadQuestion(t *testing.T) {
	q := []byte{
		0x04, 0xD2, 0x80, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, // Header
		0x01, 0x46, // F: 1 byte => F
		0x03, 0x49, 0x53, 0x49, // ISI: 3 bytes => ISI
        0x04, 0x41, 0x52, 0x50, 0x41, // ARPA: 4 bytes => ARPA
		0x00,                   // End of label
		0x03, 0x46, 0x4F, 0x4F, // FOOD: 3 bytes => FOO
		0xC0, 0x14, // Pointer to offset 20
		0xC0, 0x1A, // Pointer to offset 26
		0x0, // ROOT
        0x00, 0x01, // TYPE
        0x00, 0x01, // CLASS
	}
	expected := []DNSQuestion{
		{
			Name:  "F.ISI.ARPA",
			Class: 1,
			Type:  1,
		},
		{
			Name:  "FOO.F.ISI.ARPA",
			Class: 1,
			Type:  1,
		},
	}

	r := bytes.NewReader(q)
    // Skip the header
    r.Seek(12, io.SeekCurrent)

	parsedQuestions, err := readQuestions(r, 2)
	if err != nil {
		t.Fatalf("unexpected error reading questions: %v", err)
	}

	if !reflect.DeepEqual(expected, parsedQuestions) {
		t.Errorf("expected: %+v\nbut got: %+v\n", expected, parsedQuestions)
	}
}
