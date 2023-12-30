package main

import (
	"bytes"
	"io"
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
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

func TestReadDomain(t *testing.T) {
	tcs := []struct {
		Name                string
		buf                 []byte
		pos                 int
		expectedDomain      string
		expectByteReadCount int
	}{
		{
			Name:                "parse google.com",
			buf:                 []byte{0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00},
			pos:                 0,
			expectedDomain:      "google.com",
			expectByteReadCount: 12,
		},
		{
			Name: "parse with pointer",
			buf: []byte{
				0x01, 0x46, // F: 1 byte => F
				0x03, 0x49, 0x53, 0x49, // ISI: 3 bytes => ISI
				0x04, 0x41, 0x52, 0x50, 0x41, // ARPA: 4 bytes => ARPA
				0x00,                   // End of label
				0x03, 0x46, 0x4F, 0x4F, // FOO: 3 bytes => FOO
				0xC0, 0x00, // Pointer to offset 0 the beggining of this buffer
			},
			pos:                 12,
			expectedDomain:      "FOO.F.ISI.ARPA",
			expectByteReadCount: 6, // It doesn't include bytes read from pointer
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			r := bytes.NewReader(tc.buf)
			if tc.pos > 0 {
				_, err := r.Seek(int64(tc.pos), io.SeekStart)
				if err != nil {
					t.Fatalf("failed to seek to pos when setting up test")
				}
			}
			domain, n, err := readDomain(r, tc.pos)
			if err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}
			if n != tc.expectByteReadCount {
				t.Errorf("expected to read %d bytes but read: %d instead", tc.expectByteReadCount, n)
			}
			if domain != tc.expectedDomain {
				t.Errorf("expected domain to be %s but got %s", tc.expectedDomain, domain)
			}
		})
	}
}

func TestReadQuestion(t *testing.T) {
	tcs := []struct {
		Name                string
		buf                 []byte
		pos                 int
		expectedQuestion    DNSQuestion
		expectByteReadCount int
	}{
		{
			Name: "parse IN A google.com",
			buf: []byte{
				0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00,
				0x00, 0x01, 0x00, 0x01,
			},
			pos: 0,
			expectedQuestion: DNSQuestion{
				Name:  "google.com",
				Type:  1,
				Class: 1,
			},
			expectByteReadCount: 16,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			r := bytes.NewReader(tc.buf)
			if tc.pos > 0 {
				_, err := r.Seek(int64(tc.pos), io.SeekStart)
				if err != nil {
					t.Fatalf("failed to seek to pos when setting up test")
				}
			}
			question, n, err := readQuestion(r, tc.pos)
			if err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}
			if n != tc.expectByteReadCount {
				t.Errorf("expected to read %d bytes but read: %d instead", tc.expectByteReadCount, n)
			}
			if !reflect.DeepEqual(tc.expectedQuestion, question) {
				t.Errorf("expected domain to be %+v but got %+v", tc.expectedQuestion, question)
			}
		})
	}
}

func TestReadQuestions(t *testing.T) {
	tcs := []struct {
		Name                string
		buf                 []byte
		pos                 int
		expectedQuestions   []DNSQuestion
		expectByteReadCount int
	}{
		{
			Name: "parse IN A google.com",
			buf: []byte{
				0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00,
				0x00, 0x01, 0x00, 0x01,
			},
			pos: 0,
			expectedQuestions: []DNSQuestion{
				{
					Name:  "google.com",
					Type:  1,
					Class: 1,
				},
			},
			expectByteReadCount: 16,
		},
		{
			Name: "parse IN A F.ISI.ARPA and FOO.F.ISI.ARPA",
			buf: []byte{
				0x01, 0x46, // F: 1 byte => F
				0x03, 0x49, 0x53, 0x49, // ISI: 3 bytes => ISI
				0x04, 0x41, 0x52, 0x50, 0x41, // ARPA: 4 bytes => ARPA
				0x00,                   // End of label
				0x00, 0x01, 0x00, 0x01, // TYPE and CLASS
				0x03, 0x46, 0x4F, 0x4F, // FOO: 3 bytes => FOO
				0xC0, 0x00, // Pointer to offset 0 the beggining of this buffer
				0x00, 0x01, 0x00, 0x01, // TYPE and CLASS
			},
			pos: 0,
			expectedQuestions: []DNSQuestion{
				{
					Name:  "F.ISI.ARPA",
					Type:  1,
					Class: 1,
				},
				{
					Name:  "FOO.F.ISI.ARPA",
					Type:  1,
					Class: 1,
				},
			},
			expectByteReadCount: 26,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			r := bytes.NewReader(tc.buf)
			if tc.pos > 0 {
				_, err := r.Seek(int64(tc.pos), io.SeekStart)
				if err != nil {
					t.Fatalf("failed to seek to pos when setting up test")
				}
			}
			questions, n, err := readQuestions(r, tc.pos, uint16(len(tc.expectedQuestions)))
			if err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}
			if n != tc.expectByteReadCount {
				t.Errorf("expected to read %d bytes but read: %d instead", tc.expectByteReadCount, n)
			}
			if !reflect.DeepEqual(tc.expectedQuestions, questions) {
				t.Errorf("expected domain to be %+v but got %+v", tc.expectedQuestions, questions)
			}
		})
	}
}

func TestReadAnswer(t *testing.T) {
	tcs := []struct {
		Name                string
		buf                 []byte
		pos                 int
		expectedAnswer      DNSAnswer
		expectByteReadCount int
	}{
		{
			Name: "parse answert to IN A google.com",
			buf: []byte{
				0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, // Name: [6]google[3]com
				0x00, 0x01, // TYPE = 1
				0x00, 0x01, // CLASS = 1
				0x00, 0x00, 0x00, 0x3C, // TTL = 60
				0x00, 0x04, // RDLENGHT = 4
				0x08, 0x08, 0x08, 0x08, // DATA
			},
			pos: 0,
			expectedAnswer: DNSAnswer{
				Name:  "google.com",
				Type:  1,
				Class: 1,
				TTL:   60,
				Data:  []byte{0x08, 0x08, 0x08, 0x08},
			},
			expectByteReadCount: 26,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			r := bytes.NewReader(tc.buf)
			if tc.pos > 0 {
				_, err := r.Seek(int64(tc.pos), io.SeekStart)
				if err != nil {
					t.Fatalf("failed to seek to pos when setting up test")
				}
			}
			answer, n, err := readAnswer(r, tc.pos)
			if err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}
			if n != tc.expectByteReadCount {
				t.Errorf("expected to read %d bytes but read: %d instead", tc.expectByteReadCount, n)
			}
			if !reflect.DeepEqual(tc.expectedAnswer, answer) {
				t.Errorf("expected domain to be %+v but got %+v", tc.expectedAnswer, answer)
			}
		})
	}
}

func TestReadAnswers(t *testing.T) {
	tcs := []struct {
		Name                string
		buf                 []byte
		pos                 int
		expectedAnswers     []DNSAnswer
		expectByteReadCount int
	}{
		{
			Name: "parse answer to IN A google.com and IN A google.com",
			buf: []byte{
				0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, // Name: [6]google[3]com
				0x00, 0x01, // TYPE = 1
				0x00, 0x01, // CLASS = 1
				0x00, 0x00, 0x00, 0x3C, // TTL = 60
				0x00, 0x04, // RDLENGHT = 4
				0x08, 0x08, 0x08, 0x08, // DATA
				0x06, 0x67, 0x6f, 0x6f, 0x67, 0x6c, 0x65, 0x03, 0x63, 0x6f, 0x6d, 0x00, // Name: [6]google[3]com
				0x00, 0x01, // TYPE = 1
				0x00, 0x01, // CLASS = 1
				0x00, 0x00, 0x00, 0x3C, // TTL = 60
				0x00, 0x04, // RDLENGHT = 4
				0x08, 0x08, 0x08, 0x08, // DATA
			},
			pos: 0,
			expectedAnswers: []DNSAnswer{
				{
					Name:  "google.com",
					Type:  1,
					Class: 1,
					TTL:   60,
					Data:  []byte{0x08, 0x08, 0x08, 0x08},
				},
				{
					Name:  "google.com",
					Type:  1,
					Class: 1,
					TTL:   60,
					Data:  []byte{0x08, 0x08, 0x08, 0x08},
				},
			},
			expectByteReadCount: 52,
		},
	}

	for _, tc := range tcs {
		tc := tc
		t.Run(tc.Name, func(t *testing.T) {
			r := bytes.NewReader(tc.buf)
			if tc.pos > 0 {
				_, err := r.Seek(int64(tc.pos), io.SeekStart)
				if err != nil {
					t.Fatalf("failed to seek to pos when setting up test")
				}
			}
			answers, n, err := readAnswers(r, tc.pos, 2)
			if err != nil {
				t.Fatalf("unexpected error: %v\n", err)
			}
			if n != tc.expectByteReadCount {
				t.Errorf("expected to read %d bytes but read: %d instead", tc.expectByteReadCount, n)
			}
			if !reflect.DeepEqual(tc.expectedAnswers, answers) {
				t.Errorf("expected domain to be %+v but got %+v", tc.expectedAnswers, answers)
			}
		})
	}
}

func TestDNSMessage_MarshalBinary(t *testing.T) {
	reqBytes := []byte{
		0xb1, 0x87, 0x01, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x0c, 0x63, 0x6f, 0x64, 0x65, 0x63, 0x72, 0x61, 0x66, 0x74, 0x65, 0x72, 0x73,
		0x02, 0x69, 0x6f,
		0x00,
		0x00, 0x01, 0x00, 0x01,
	}
	expectedMessage := DNSMessage{
		Header: DNSHeader{
			ID: 45447,
			Flags: DNSHeaderFlags{
				RD: true,
			},
			QDCOUNT: 1,
		},
		Questions: []DNSQuestion{
			{
				Name:  "codecrafters.io",
				Type:  1,
				Class: 1,
			},
		},
		Answers: []DNSAnswer{},
	}

	var request DNSMessage
	if err := request.UnmarshalBinary(reqBytes); err != nil {
		t.Fatalf("failed to unmarshal request: %v", err)
	}
	if !cmp.Equal(expectedMessage, request) {
		t.Errorf("result does not match expected output: %s", cmp.Diff(expectedMessage, request))
	}
}
